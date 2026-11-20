//go:build windows

package main

// Development-only input runtime. The TS Extra style physical plugin remains
// gated until its ATS 1.62 layout is validated in a controlled game test.

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var epUser32 = syscall.NewLazyDLL("user32.dll")
var getAsyncKey = epUser32.NewProc("GetAsyncKeyState")
var getForeground = epUser32.NewProc("GetForegroundWindow")
var getWindowPID = epUser32.NewProc("GetWindowThreadProcessId")
var openProcess = epKernel32.NewProc("OpenProcess")
var queryImage = epKernel32.NewProc("QueryFullProcessImageNameW")

func isForegroundATS() bool {
	hwnd, _, _ := getForeground.Call()
	if hwnd == 0 {
		return false
	}
	var pid uint32
	getWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return false
	}
	h, _, _ := openProcess.Call(0x1000, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer epCloseHandle.Call(h)
	var name [1024]uint16
	length := uint32(len(name))
	ret, _, _ := queryImage.Call(h, 0, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&length)))
	if ret == 0 {
		return false
	}
	return strings.EqualFold(filepath.Base(syscall.UTF16ToString(name[:length])), "amtrucks.exe")
}

func codeToVK(binding string) uint16 {
	if !strings.HasPrefix(binding, "KEY:") {
		return 0
	}
	code := strings.TrimPrefix(binding, "KEY:")
	if strings.HasPrefix(code, "Key") && len(code) == 4 && code[3] >= 'A' && code[3] <= 'Z' {
		return uint16(code[3])
	}
	if strings.HasPrefix(code, "Digit") && len(code) == 6 && code[5] >= '0' && code[5] <= '9' {
		return uint16(code[5])
	}
	if strings.HasPrefix(code, "Numpad") && len(code) == 7 && code[6] >= '0' && code[6] <= '9' {
		return uint16(0x60 + code[6] - '0')
	}
	if strings.HasPrefix(code, "F") {
		if x, e := strconv.Atoi(code[1:]); e == nil && x >= 1 && x <= 24 {
			return 0x70 + uint16(x) - 1
		}
	}
	return map[string]uint16{"Space": 0x20, "ArrowLeft": 0x25, "ArrowUp": 0x26, "ArrowRight": 0x27, "ArrowDown": 0x28, "Enter": 0x0D, "Escape": 0x1B, "ShiftLeft": 0xA0, "ShiftRight": 0xA1, "ControlLeft": 0xA2, "ControlRight": 0xA3, "AltLeft": 0xA4, "AltRight": 0xA5, "Insert": 0x2D, "Delete": 0x2E}[code]
}
func deviceBindingDown(binding string, devices []device) bool {
	if strings.HasPrefix(binding, "KEY:") {
		vk := codeToVK(binding)
		if vk == 0 {
			return false
		}
		state, _, _ := getAsyncKey.Call(uintptr(vk))
		return state&0x8000 != 0
	}
	parts := strings.Split(binding, ":")
	if len(parts) < 4 || parts[0] != "DEVICE" {
		return false
	}
	for _, d := range devices {
		if d.ID != parts[1] {
			continue
		}
		idx, err := strconv.Atoi(parts[3])
		if err != nil {
			return false
		}
		if parts[2] == "BUTTON" && idx >= 0 && idx < len(d.Buttons) {
			return d.Buttons[idx]
		}
		if parts[2] == "POV" {
			return d.POV == idx
		}
		if parts[2] == "AXIS" && len(parts) == 5 && idx >= 0 && idx < len(d.Axes) {
			if parts[4] == "+" {
				return d.Axes[idx] > 0.6
			}
			if parts[4] == "-" {
				return d.Axes[idx] < -0.6
			}
		}
	}
	return false
}

type runtimeAxle struct {
	Enabled    bool    `json:"enabled"`
	Reversed   bool    `json:"reversed"`
	MaxAngle   float64 `json:"max_angle"`
	TurnRate   float64 `json:"turn_rate"`
	CenterRate float64 `json:"center_rate"`
	Ratio      float64 `json:"ratio"`
}
type runtimeTrailer struct {
	ID    string        `json:"id"`
	Axles []runtimeAxle `json:"axles"`
}
type runtimeProfile struct {
	Name             string           `json:"name"`
	ID               string           `json:"id"`
	MatchKey         string           `json:"match_key"`
	ToggleKey        string           `json:"toggle_key"`
	LeftKey          string           `json:"left_key"`
	RightKey         string           `json:"right_key"`
	AnalogAxis       string           `json:"analog_axis"`
	AnalogAxisCenter float64          `json:"analog_axis_center"`
	AnalogInvert     bool             `json:"analog_invert"`
	AnalogDeadzone   float64          `json:"analog_deadzone"`
	AnalogCurve      float64          `json:"analog_curve"`
	AnalogFullScale  float64          `json:"analog_full_scale"`
	AnalogOutputPct  float64          `json:"analog_output_pct"`
	AutoCenter       bool             `json:"auto_center_moving"`
	Trailers         []runtimeTrailer `json:"trailers"`
}
type runtimeProfiles struct {
	ActiveID string           `json:"active_id"`
	Profiles []runtimeProfile `json:"profiles"`
}

func findProfile(p runtimeProfiles, trailerID string) (runtimeProfile, bool) {
	for _, candidate := range p.Profiles {
		if candidate.MatchKey != "" && strings.EqualFold(candidate.MatchKey, trailerID) {
			return candidate, true
		}
	}
	for _, candidate := range p.Profiles {
		if candidate.ID == p.ActiveID && candidate.MatchKey == "" {
			return candidate, true
		}
	}
	return runtimeProfile{}, false
}

// The game bridge is per trailer; reject conflicting per-axle directions,
// so the UI never claims to control different axle groups when it can't.
func trailerCompatible(t runtimeTrailer) (runtimeAxle, bool) {
	var chosen runtimeAxle
	found := false
	for _, a := range t.Axles {
		if !a.Enabled {
			continue
		}
		if !found {
			chosen = a
			found = true
			continue
		}
		if a.Reversed != chosen.Reversed || math.Abs(a.MaxAngle-chosen.MaxAngle) > 0.01 || math.Abs(a.Ratio-chosen.Ratio) > 0.01 {
			return runtimeAxle{}, false
		}
	}
	return chosen, found && chosen.MaxAngle > 0 && chosen.MaxAngle <= 60 && chosen.TurnRate >= 0 && chosen.TurnRate <= 90 && chosen.CenterRate >= 0 && chosen.CenterRate <= 90
}
func analogFromDevices(p runtimeProfile, devices []device) float64 {
	parts := strings.Split(p.AnalogAxis, ":")
	if len(parts) != 4 || parts[0] != "DEVICE" || parts[2] != "AXIS" {
		return 0
	}
	idx, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0
	}
	for _, d := range devices {
		if d.ID == parts[1] && idx >= 0 && idx < len(d.Axes) {
			raw := d.Axes[idx]
			neutral := math.Max(-.95, math.Min(.95, p.AnalogAxisCenter))
			centered := 0.
			if raw >= neutral {
				centered = (raw - neutral) / (1 - neutral)
			} else {
				centered = (raw - neutral) / (1 + neutral)
			}
			deadzone := math.Max(0, math.Min(.4, p.AnalogDeadzone))
			if math.Abs(centered) <= deadzone {
				return 0
			}
			curve := math.Max(.5, math.Min(3, p.AnalogCurve))
			if curve == 0 {
				curve = 1
			}
			scale := p.AnalogFullScale
			if scale <= 0 {
				scale = 100
			}
			scale = math.Max(20, math.Min(100, scale))
			output := p.AnalogOutputPct
			if output < 0 || output > 100 {
				output = 100
			}
			value := math.Copysign(math.Pow(math.Min(1, math.Abs(centered)*100/scale), curve), centered) * output / 100
			if p.AnalogInvert {
				value = -value
			}
			return value
		}
	}
	return 0
}

// Stage-1 input routing: read the saved active profile every 250ms.
// Uses the same mapping format as the real physical runtime: KEY:* and
// DEVICE:*; the physical DLL never hardcodes F8/F9/F10.
func startSteeringRuntime(s *server) func() {
	bridge, bridgeErr := newPhysicalSteeringBridge()
	if bridgeErr != nil {
		log.Printf("EssieuPilot IPC preparation unavailable: %v", bridgeErr)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if bridge != nil {
			defer bridge.close()
		}
		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()
		var profiles runtimeProfiles
		var lastProfileRead time.Time
		var lastInput string
		var lastEventTime time.Time
		var manual, prevToggle, prevLeft, prevRight, prevAnalog bool
		var profileID string
		var profileLoaded bool
		var gameRunning bool
		var activationCount, leftCount, rightCount uint64
		var lastDirection string
		var lastDirectionTime time.Time
		var lastFocusTime time.Time
		var history []string
		addHistory := func(event string, at time.Time) {
			history = append(history, at.Format("15:04:05")+" "+event)
			if len(history) > 6 { history = append([]string(nil), history[len(history)-6:]...) }
		}
		for {
			select {
			case <-ctx.Done():
				if bridge != nil {
					_ = bridge.publish(false, 0, 0)
				}
				publishInputLinkSnapshot(inputLinkSnapshot{})
				return
			case <-ticker.C:
			}
			now := time.Now()
			if now.Sub(lastProfileRead) >= 250*time.Millisecond {
				lastProfileRead = now
				gameRunning = statusATSRunning()
				buf, err := s.store.read()
				var current runtimeProfiles
				profileLoaded = err == nil && json.Unmarshal(buf, &current) == nil && len(current.Profiles) > 0
				if profileLoaded {
					profiles = current
				} else {
					profiles = runtimeProfiles{}
				}
			}
			active, found := activeSavedProfile(profiles)
			if !found {
				profileID = ""
			}
			if active.ID != profileID {
				profileID = active.ID
				manual = false
				prevToggle = false
				prevLeft = false
				prevRight = false
				prevAnalog = false
				lastInput = ""
				lastEventTime = time.Time{}
				lastDirection = ""
				lastDirectionTime = time.Time{}
				activationCount, leftCount, rightCount = 0, 0, 0
				history = nil
			}
			manual = retainManualOnFocusChange(manual, gameRunning)
			foreground := gameRunning && isForegroundATS()
			if foreground {lastFocusTime = now}
			snapshot := inputLinkSnapshot{
				ServiceRunning: true, IPCReady: bridge != nil,
				ProfileSaved:           profileLoaded && found,
				ProfileName:            active.Name,
				ToggleBinding:          active.ToggleKey,
				LeftBinding:            active.LeftKey,
				RightBinding:           active.RightKey,
				AnalogBinding:          active.AnalogAxis,
				ATSForeground:          foreground,
				ATSRunning:             gameRunning,
				PhysicalControlEnabled: false,
				DLLConfirmed:           false,
				StatusMessage:          "Touches transmises au contrôleur. Canal IPC neutre : aucun braquage physique activé.",
			}
			if !foreground {
				// ATS lost focus: do not read input from another application.
				// Keep the last armed/unarmed state for the Diagnostic page. Old code
				// reset manual=false on every Alt+Tab, hiding successful activation.
				prevToggle = false
				prevLeft = false
				prevRight = false
				prevAnalog = false
			} else if found {
				devices := readDevices()
				toggle := deviceBindingDown(active.ToggleKey, devices)
				if risingEdge(toggle, prevToggle) {
					manual = !manual
					activationCount++
					lastInput = "Activation (" + active.ToggleKey + ")"
					if !manual { lastInput = "Désactivation (" + active.ToggleKey + ")" }
					lastEventTime = now
					addHistory(lastInput, now)
				}
				prevToggle = toggle
				left := deviceBindingDown(active.LeftKey, devices)
				right := deviceBindingDown(active.RightKey, devices)
				// Always record a raw press. When manual mode is off, label it
				// explicitly; otherwise the user cannot distinguish bad bindings
				// from a disabled steering toggle.
				if risingEdge(left, prevLeft) {
					leftCount++
					lastInput = "Gauche (" + active.LeftKey + ")"
					if !manual {lastInput += " — pilotage désactivé"}
					lastEventTime = now
					lastDirection = lastInput
					lastDirectionTime = now
					addHistory(lastInput, now)
				}
				if risingEdge(right, prevRight) {
					rightCount++
					lastInput = "Droite (" + active.RightKey + ")"
					if !manual {lastInput += " — pilotage désactivé"}
					lastEventTime = now
					lastDirection = lastInput
					lastDirectionTime = now
					addHistory(lastInput, now)
				}
				prevLeft = left
				prevRight = right
				analog := math.Abs(analogFromDevices(active, devices)) >= 0.1
				if risingEdge(analog, prevAnalog) {
					lastInput = "Axe analogique (" + active.AnalogAxis + ")"
					if !manual {lastInput += " — pilotage désactivé"}
					lastEventTime = now
					addHistory(lastInput, now)
				}
				prevAnalog = analog
				snapshot.LeftPressed = left && manual
				snapshot.RightPressed = right && manual
			}
			snapshot.ManualArmed = manual
			snapshot.ActivationCount = activationCount
			snapshot.LeftCount = leftCount
			snapshot.RightCount = rightCount
			snapshot.LastDirection = lastDirection
			_, snapshot.LastDirectionTime = formatInputEvent(lastDirection, lastDirectionTime)
			if !lastFocusTime.IsZero() {snapshot.LastFocusTime = lastFocusTime.Format(time.RFC3339)}
			snapshot.InputHistory = append([]string(nil), history...)
			snapshot.LastInput, snapshot.LastEventTime = formatInputEvent(lastInput, lastEventTime)
			// Important safety interlock: NEVER send an enabled packet to a
			// version-unverified DLL. This first update tests mapping + IPC only.
			if bridge != nil {
				if err := bridge.publish(false, 0, 0); err != nil {
					snapshot.IPCReady = false
					snapshot.StatusMessage = "Canal IPC indisponible : " + err.Error()
				}
			}
			publishInputLinkSnapshot(snapshot)
		}
	}()
	return func() { cancel(); <-done }
}

// Only leaving the ATS process entirely disarms the diagnostic. Switching
// focus to EssieuPilot does not reset activation: the user must be able to
// inspect the prior state after Alt+Tab.
func retainManualOnFocusChange(manual, gameRunning bool) bool {
	return manual && gameRunning
}

// risingEdge records one press per key-down transition. It does not
// presume that steering is armed, so disabled-input tests remain observable.
func risingEdge(pressed, previouslyPressed bool) bool {
	return pressed && !previouslyPressed
}

func activeSavedProfile(profiles runtimeProfiles) (runtimeProfile, bool) {
	for _, candidate := range profiles.Profiles {
		if candidate.ID == profiles.ActiveID {
			return candidate, true
		}
	}
	return runtimeProfile{}, false
}
