//go:build windows

package main

import (
	"fmt"
	"math"
	"syscall"
	"unsafe"
)

// Windows-specific device reader, compatible with frontend DEVICE:<id>:AXIS:<n> bindings.
// XInput uses the native gamepad API; WinMM supports legacy joysticks/wheels.
// This is read-only input capture: it does NOT steer axles in ATS.
var winmm = syscall.NewLazyDLL("winmm.dll")
var joyGetNumDevs = winmm.NewProc("joyGetNumDevs")
var joyGetPosEx = winmm.NewProc("joyGetPosEx")

type joyInfo struct {
	Size, Flags                                      uint32
	X, Y, Z, R, U, V                                 uint32
	Buttons, ButtonNumber, POV, Reserved1, Reserved2 uint32
}

func scale(v uint32) float64 { return math.Max(-1, math.Min(1, (float64(v)-32767.5)/32767.5)) }

func readDevices() []device {
	result := make([]device, 0, 16)
	for idx := 0; idx < 4; idx++ {
		if d, ok := readXInput(idx); ok {
			result = append(result, d)
		}
	}
	n, _, _ := joyGetNumDevs.Call()
	if n > 16 {
		n = 16
	}
	for i := uintptr(0); i < n; i++ {
		info := joyInfo{Size: uint32(unsafe.Sizeof(joyInfo{})), Flags: 0xFF}
		rv, _, _ := joyGetPosEx.Call(i, uintptr(unsafe.Pointer(&info)))
		if rv != 0 {
			continue
		}
		buttons := make([]bool, 32)
		for j := 0; j < 32; j++ {
			buttons[j] = info.Buttons&(1<<uint(j)) != 0
		}
		axes := []float64{scale(info.X), scale(info.Y), scale(info.Z), scale(info.R), scale(info.U), scale(info.V)}
		pov := -1
		if info.POV != 0xFFFF && info.POV != 0xFFFFFFFF {
			pov = int(info.POV / 4500)
		}
		result = append(result, device{ID: fmt.Sprintf("JOY%d", i), Name: fmt.Sprintf("Joystick / volant Windows %d", i+1), Buttons: buttons, Axes: axes, POV: pov})
	}
	return result
}

type xinputPad struct {
	Buttons                            uint16
	LeftTrigger, RightTrigger          uint8
	ThumbLX, ThumbLY, ThumbRX, ThumbRY int16
}
type xinputState struct {
	Packet uint32
	Pad    xinputPad
}

var xinputCandidates = []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"}

func readXInput(index int) (device, bool) {
	for _, name := range xinputCandidates {
		dll := syscall.NewLazyDLL(name)
		proc := dll.NewProc("XInputGetState")
		if err := proc.Find(); err != nil {
			continue
		}
		var st xinputState
		rv, _, _ := proc.Call(uintptr(index), uintptr(unsafe.Pointer(&st)))
		if rv != 0 {
			return device{}, false
		}
		btn := make([]bool, 16)
		for j := 0; j < 16; j++ {
			btn[j] = st.Pad.Buttons&(1<<uint(j)) != 0
		}
		axes := []float64{float64(st.Pad.ThumbLX) / 32768, float64(st.Pad.ThumbLY) / 32768, float64(st.Pad.ThumbRX) / 32768, float64(st.Pad.ThumbRY) / 32768, float64(st.Pad.LeftTrigger) / 255, float64(st.Pad.RightTrigger) / 255}
		return device{ID: fmt.Sprintf("XINPUT%d", index), Name: fmt.Sprintf("Manette XInput %d", index+1), Buttons: btn, Axes: axes, POV: -1}, true
	}
	return device{}, false
}
