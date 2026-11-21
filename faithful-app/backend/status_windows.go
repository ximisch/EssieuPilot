//go:build windows

package main

import (
    "encoding/binary"
    "strings"
    "syscall"
    "unsafe"
)

const (
    snapProcess = 2
    fileMapRead = 0x0004
)

var (
    statusKernel = syscall.NewLazyDLL("kernel32.dll")
    createSnapshot = statusKernel.NewProc("CreateToolhelp32Snapshot")
    processFirstW = statusKernel.NewProc("Process32FirstW")
    processNextW = statusKernel.NewProc("Process32NextW")
    closeStatusHandle = statusKernel.NewProc("CloseHandle")
    openStatusMap = statusKernel.NewProc("OpenFileMappingW")
    mapStatusView = statusKernel.NewProc("MapViewOfFile")
    unmapStatusView = statusKernel.NewProc("UnmapViewOfFile")
    statusTickCount = statusKernel.NewProc("GetTickCount64")
)
type statusProcessEntry struct {
    Size uint32
    Usage uint32
    PID uint32
    Heap uintptr
    ModuleID uint32
    Threads uint32
    ParentPID uint32
    BasePriority int32
    Flags uint32
    Exe [260]uint16
}

func statusATSRunning() bool {
    handle, _, _ := createSnapshot.Call(snapProcess, 0)
    if handle == ^uintptr(0) || handle == 0 { return false }
    defer closeStatusHandle.Call(handle)
    entry := statusProcessEntry{Size: uint32(unsafe.Sizeof(statusProcessEntry{}))}
    ok, _, _ := processFirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
    for ok != 0 {
        if strings.EqualFold(syscall.UTF16ToString(entry.Exe[:]), "amtrucks.exe") {return true}
        ok, _, _ = processNextW.Call(handle,uintptr(unsafe.Pointer(&entry)))
    }
    return false
}

// Only DLL acknowledgement emitted after a steering hook calls the game routine.
// A recent ACK is not evidence that the axle model visibly moves in ATS.
func statusSteeringACK() bool {
    name, _ := syscall.UTF16PtrFromString(`Local\EssieuPilotAppliedV1`)
    handle, _, _ := openStatusMap.Call(fileMapRead, 0, uintptr(unsafe.Pointer(name)))
    if handle == 0 { return false }
    defer closeStatusHandle.Call(handle)
    view, _, _ := mapStatusView.Call(handle,fileMapRead,0,0,32)
    if view == 0 {return false}
    defer unmapStatusView.Call(view)
    var buf [32]byte
    seq1 := binary.LittleEndian.Uint32(unsafe.Slice((*byte)(unsafe.Pointer(view+8)),4))
    copy(buf[:],unsafe.Slice((*byte)(unsafe.Pointer(view)),32))
    seq2 := binary.LittleEndian.Uint32(unsafe.Slice((*byte)(unsafe.Pointer(view+8)),4))
    magic:=binary.LittleEndian.Uint32(buf[0:4])
    version:=binary.LittleEndian.Uint32(buf[4:8])
    begin:=binary.LittleEndian.Uint32(buf[8:12])
    end:=binary.LittleEndian.Uint32(buf[12:16])
    timestamp:=binary.LittleEndian.Uint64(buf[16:24])
    tick,_,_:=statusTickCount.Call()
    now:=uint64(tick)
    return magic==0x45504131 && version==1 && begin==end &&
        begin==seq1 && begin==seq2 && begin&1==0 &&
        now>=timestamp && now-timestamp<=1500
}
