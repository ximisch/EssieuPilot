//go:build windows

package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const steeringMapName = `Local\EssieuPilotSteeringV1`

var epKernel32 = syscall.NewLazyDLL("kernel32.dll")
var epCreateMap = epKernel32.NewProc("CreateFileMappingW")
var epMapView = epKernel32.NewProc("MapViewOfFile")
var epUnmapView = epKernel32.NewProc("UnmapViewOfFile")
var epCloseHandle = epKernel32.NewProc("CloseHandle")
var epUptime = epKernel32.NewProc("GetTickCount64")

type physicalSteeringBridge struct {
	handle, view uintptr
	seq          uint32
	mu           sync.Mutex
}

func newPhysicalSteeringBridge() (*physicalSteeringBridge, error) {
	name, e := syscall.UTF16PtrFromString(steeringMapName)
	if e != nil {
		return nil, e
	}
	// INVALID_HANDLE_VALUE maps anonymous, page-file-backed local memory.
	h, _, callErr := epCreateMap.Call(^uintptr(0), 0, 4, 0, steeringPacketSize, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return nil, callErr
	}
	view, _, err := epMapView.Call(h, 0x0002|0x0004, 0, 0, steeringPacketSize)
	if view == 0 {
		epCloseHandle.Call(h)
		return nil, err
	}
	return &physicalSteeringBridge{handle: h, view: view}, nil
}
func (b *physicalSteeringBridge) publish(enabled bool, index uint32, angle float32) error {
	if b == nil {
		return errors.New("bridge not initialized")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.view == 0 {
		return errors.New("bridge closed")
	}
	if !enabled {
		index = 0
		angle = 0
	}
	seq := b.seq + 2
	uptime, _, _ := epUptime.Call()
	p := steeringPacket{Seq: seq, Enabled: enabled, UptimeMs: uint64(uptime), TrailerIndex: index, NormalizedAngle: angle}
	var raw [steeringPacketSize]byte
	if err := encodeSteeringPacket(raw[:], p); err != nil {
		return err
	}
	view := unsafe.Slice((*byte)(unsafe.Pointer(b.view)), steeringPacketSize)
	begin := (*uint32)(unsafe.Pointer(b.view + 8))
	end := (*uint32)(unsafe.Pointer(b.view + 32))
	atomic.StoreUint32(begin, seq-1)
	copy(view[0:8], raw[0:8])
	copy(view[12:32], raw[12:32])
	atomic.StoreUint32(end, seq)
	atomic.StoreUint32(begin, seq)
	b.seq = seq
	return nil
}
func (b *physicalSteeringBridge) close() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.view != 0 {
		epUnmapView.Call(b.view)
		b.view = 0
	}
	if b.handle != 0 {
		epCloseHandle.Call(b.handle)
		b.handle = 0
	}
}
