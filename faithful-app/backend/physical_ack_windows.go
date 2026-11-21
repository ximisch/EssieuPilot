//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const physicalAckMapName = `Local\EssieuPilotAppliedV1`

// The DLL creates this map only after it invokes the in-game trailer
// steering callback. Acknowledgement is not evidence of visual wheel motion.
func readPhysicalAck() physicalAckReport {
	name,err:=syscall.UTF16PtrFromString(physicalAckMapName)
	if err!=nil {return physicalAckReport{}}
	const fileMapRead=0x0004
	h,_,_:=epOpenFileMapping.Call(fileMapRead,0,uintptr(unsafe.Pointer(name)))
	if h==0 {return physicalAckReport{}}
	defer epCloseHandle.Call(h)
	view,_,_:=epMapView.Call(h,fileMapRead,0,0,physicalAckSize)
	if view==0{return physicalAckReport{}}
	raw:=make([]byte,physicalAckSize)
	copy(raw,unsafe.Slice((*byte)(unsafe.Pointer(view)),physicalAckSize))
	epUnmapView.Call(view)
	uptime,_,_:=epUptime.Call()
	report,err:=decodePhysicalAck(raw,uint64(uptime))
	if err!=nil{return physicalAckReport{}}
	return report
}
