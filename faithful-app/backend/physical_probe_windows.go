//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const physicalProbeMapName = `Local\EssieuPilotProbeV8`

var epOpenFileMapping = epKernel32.NewProc("OpenFileMappingW")

// Read-only diagnostic. The memory mapping belongs to the DLL and disappears
// when ATS unloads it. The controller never treats this as physics validation.
func readPhysicalDLLProbe() physicalProbeReport {
	fail := physicalProbeReport{Reason: "Aucun rapport de chargement de la DLL dans ATS"}
	name, err := syscall.UTF16PtrFromString(physicalProbeMapName)
	if err != nil {
		fail.Reason = "Nom du canal DLL invalide"
		return fail
	}
	const fileMapRead = 0x0004
	h, _, _ := epOpenFileMapping.Call(fileMapRead, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return fail
	}
	defer epCloseHandle.Call(h)
	view, _, _ := epMapView.Call(h, fileMapRead, 0, 0, physicalProbeSize)
	if view == 0 {
		fail.Reason = "DLL présente, lecture du rapport impossible"
		return fail
	}
	bytes := make([]byte, physicalProbeSize)
	copy(bytes, unsafe.Slice((*byte)(unsafe.Pointer(view)), physicalProbeSize))
	epUnmapView.Call(view)
	uptime, _, _ := epUptime.Call()
	result, err := decodePhysicalProbe(bytes, uint64(uptime))
	if err != nil {
		fail.Reason = "Rapport DLL invalide : " + err.Error()
		return fail
	}
	return result
}
