//go:build windows
package main

import (
    "syscall"
    "unsafe"
)
const convoyProbeMapName = `Local\EssieuPilotConvoyProbeV1`

// Read-only local shared memory published by EssieuPilotATS.dll.
// No changes to game memory, ATS server traffic or remote player physics.
func readConvoyProbe() convoyProbeReport {
    name,err:=syscall.UTF16PtrFromString(convoyProbeMapName)
    if err!=nil {return unknownConvoyProbe("identifiant de sonde invalide")}
    const fileMapRead=0x0004
    handle,_,_:=epOpenFileMapping.Call(fileMapRead,0,uintptr(unsafe.Pointer(name)))
    if handle==0 {return unknownConvoyProbe("DLL Original 21 non chargée ou ATS fermé")}
    defer epCloseHandle.Call(handle)
    view,_,_:=epMapView.Call(handle,fileMapRead,0,0,convoyProbeSize)
    if view==0 {return unknownConvoyProbe("lecture du canal Windows impossible")}
    var raw [convoyProbeSize]byte
    copy(raw[:],unsafe.Slice((*byte)(unsafe.Pointer(view)),convoyProbeSize))
    epUnmapView.Call(view)
    uptime,_,_:=epUptime.Call()
    result,err:=decodeConvoyProbe(raw[:],uint64(uptime))
    if err!=nil{return unknownConvoyProbe(err.Error())}
    return result
}
