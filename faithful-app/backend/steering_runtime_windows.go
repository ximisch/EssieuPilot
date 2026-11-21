//go:build windows

package main

// Development-only input runtime. The TS Extra style physical plugin remains
// gated until its ATS 1.62 layout is validated in a controlled game test.

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
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

// Input capture is gated to the actual foreground ATS process. Windows can
// deny OpenProcess/QueryFullProcessImageName for elevated games even when
// Toolhelp32 is able to enumerate amtrucks.exe. Prefer PID matching first.
type atsFocusInfo struct {
	IsATS bool
	ForegroundPID uint32
	ATSProcessPID uint32
	Reason string
}

func atsProcessIDs() []uint32 {
	handle, _, _ := createSnapshot.Call(snapProcess, 0)
	if handle == ^uintptr(0) || handle == 0 { return nil }
	defer closeStatusHandle.Call(handle)
	entry := statusProcessEntry{Size: uint32(unsafe.Sizeof(statusProcessEntry{}))}
	ok, _, _ := processFirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	var ids []uint32
	for ok != 0 {
		if strings.EqualFold(syscall.UTF16ToString(entry.Exe[:]), "amtrucks.exe") {
			ids = append(ids, entry.PID)
		}
		entry.Size = uint32(unsafe.Sizeof(statusProcessEntry{}))
		ok, _, _ = processNextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	}
	return ids
}

func detectATSForeground() atsFocusInfo {
	hwnd, _, _ := getForeground.Call()
	if hwnd == 0 {
		return atsFocusInfo{Reason: "Fenêtre active Windows indisponible"}
	}
	var pid uint32
	getWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return atsFocusInfo{Reason: "PID de la fenêtre active indisponible"}
	}
	result := atsFocusInfo{ForegroundPID:pid}
	ids := atsProcessIDs()
	if len(ids) > 0 { result.ATSProcessPID = ids[0] }
	if foregroundMatchesATS(pid, ids) {
		result.IsATS = true
		result.ATSProcessPID = pid
		result.Reason = "Fenêtre ATS reconnue par son PID"
		return result
	}
	// Fallback for Windows builds where Process32 enumeration is restricted.
	h, _, _ := openProcess.Call(0x1000, 0, uintptr(pid))
	if h == 0 {
		result.Reason = "Fenêtre active non reconnue ; lecture du PID refusée par Windows"
		return result
	}
	defer epCloseHandle.Call(h)
	var name [1024]uint16
	length := uint32(len(name))
	ret, _, _ := queryImage.Call(h, 0, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&length)))
	if ret == 0 {
		result.Reason = "Impossible de lire le nom du processus actif"
		return result
	}
	if strings.EqualFold(filepath.Base(syscall.UTF16ToString(name[:length])), "amtrucks.exe") {
		result.IsATS = true
		result.ATSProcessPID = pid
		result.Reason = "Fenêtre ATS reconnue par le nom du processus"
		return result
	}
	result.Reason = "Une autre application est au premier plan"
	return result
}

func foregroundMatchesATS(activePID uint32, gamePIDs []uint32) bool {
	if activePID == 0 { return false }
	for _, gamePID := range gamePIDs {
		if gamePID != 0 && activePID == gamePID { return true }
	}
	return false
}

func isForegroundATS() bool { return detectATSForeground().IsATS }

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
	return map[string]uint16{
		"Space": 0x20, "Tab": 0x09, "Backspace": 0x08, "Enter": 0x0D, "NumpadEnter": 0x0D,
		"Escape": 0x1B, "CapsLock": 0x14, "Pause": 0x13, "NumLock": 0x90,
		"ScrollLock": 0x91, "PrintScreen": 0x2C,
		"ArrowLeft": 0x25, "ArrowUp": 0x26, "ArrowRight": 0x27, "ArrowDown": 0x28,
		"PageUp": 0x21, "PageDown": 0x22, "End": 0x23, "Home": 0x24,
		"Insert": 0x2D, "Delete": 0x2E,
		"ShiftLeft": 0xA0, "ShiftRight": 0xA1, "ControlLeft": 0xA2,
		"ControlRight": 0xA3, "AltLeft": 0xA4, "AltRight": 0xA5,
		"NumpadAdd": 0x6B, "NumpadSubtract": 0x6D,
		"NumpadMultiply": 0x6A, "NumpadDivide": 0x6F, "NumpadDecimal": 0x6E,
		"Semicolon": 0xBA, "Equal": 0xBB, "Comma": 0xBC, "Minus": 0xBD,
		"Period": 0xBE, "Slash": 0xBF, "Backquote": 0xC0,
		"BracketLeft": 0xDB, "Backslash": 0xDC, "BracketRight": 0xDD, "Quote": 0xDE,
	}[code]
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
	// When set, bind this saved configuration to the exact ATS trailer
	// telemetry identifier. Never rely on preset index or the UI name.
	ATSMatchKey string `json:"ats_match_key"`
	// Number of requested (nominal) degrees corresponding to an ATS
	// normalized command of 1.0. Default 60; calibrate on an observed
	// trailer, NOT an assertion about physical wheel angles.
	PhysicalFullScaleDegrees float64 `json:"physical_full_scale_degrees"`
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
	// Optional explicit binding of ONE actual ATS trailer to a saved
	// configuration. Supports any count of presets, never assumes index 0.
	PhysicalSelectedTrailerID string `json:"physical_selected_trailer_id"`
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

// Explicitly resolve ONE selected configuration for the single attached
// trailer supported by the current experimental DLL. Other configurations
// remain stored and untouched. For multiple presets, never silently send a
// potentially wrong trailer's steering data to the game.
func selectedPhysicalTrailer(p runtimeProfile) (runtimeTrailer, bool) {
    if len(p.Trailers)==1 {
        if p.PhysicalSelectedTrailerID!="" && p.PhysicalSelectedTrailerID!=p.Trailers[0].ID {
            return runtimeTrailer{},false
        }
        return p.Trailers[0],true
    }
    if p.PhysicalSelectedTrailerID=="" {return runtimeTrailer{},false}
    for _,t:=range p.Trailers {
        if t.ID==p.PhysicalSelectedTrailerID {return t,true}
    }
    return runtimeTrailer{},false
}

// Resolve the attached trailer ONLY by a unique exact saved ATS identity.
// Unmatched / duplicated explicit mappings fail closed (never use an old
// selection from another trailer). Old unbound profiles retain their manual
// Original.13 selection until a user creates an explicit association.
func selectedPhysicalTrailerForATS(p runtimeProfile, detectedID string) (runtimeTrailer,string,bool) {
    id:=strings.TrimSpace(detectedID)
    if id=="" {return runtimeTrailer{},"TELEMETRIE_REMORQUE_SANS_IDENTIFIANT",false}
    var exact runtimeTrailer
    matches:=0
    hasLinks:=false
    for _,t:=range p.Trailers {
        if strings.TrimSpace(t.ATSMatchKey)==""{continue}
        hasLinks=true
        if strings.EqualFold(strings.TrimSpace(t.ATSMatchKey),id) {
            exact=t
            matches++
        }
    }
    if matches>1{return runtimeTrailer{},"IDENTIFIANT_REMORQUE_AMBIGU",false}
    if matches==1{return exact,"ASSOCIATION_AUTOMATIQUE_EXACTE",true}
    if hasLinks{return runtimeTrailer{},"REMORQUE_INCONNUE_ASSOCIATION_REQUISE",false}
    legacy,ok:=selectedPhysicalTrailer(p)
    if !ok{return runtimeTrailer{},"CONFIG_REMORQUE_A_SELECTIONNER",false}
    return legacy,"SELECTION_MANUELLE_NON_ASSOCIEE",true
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

// A release build is always safe by default. R&D-specific command delivery
// requires both an explicitly customized build and an explicit runtime opt-in.
// Never set this to "true" in the public update or installer pipeline.
var physicalResearchBuildMode = "true"

// Experimental Original.10 only. The installer never enables test steering
// automatically: the user must explicitly choose the solo crash-risk task.
// The consent marker belongs to this isolated test, NOT stable EssieuPilot.
func physicalTestConsent() bool {
  roaming, err := os.UserConfigDir()
  if err != nil || roaming == "" { return false }
  path := filepath.Join(roaming, "EssieuPilot-OriginalTest", "physical-test-consent.txt")
  content, err := os.ReadFile(path)
  return err == nil && strings.TrimSpace(string(content)) == "I_ACCEPT_CRASH_RISK"
}

// Original.7 input routing: re-read the saved active profile every 250ms.
// Uses the same mapping format as the real physical runtime: KEY:* and
// DEVICE:*; the physical DLL never hardcodes F8/F9/F10.
func startSteeringRuntime(s *server) func() {
	// Only a manually launched R&D executable with this *exact* opt-in may
	// send an experimental steering command. Never enable in release builds
	// before ATS binary offsets and actual wheel movement are verified.
	physicalResearchOptIn := physicalResearchBuildMode == "true" && physicalTestConsent()
	// Original.12 records the last IN-GAME gate result. Focus loss must never
	// overwrite evidence captured while controls were active in ATS.
	diagnostic := newPhysicalDiagnosticRecorder()
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
		// Isolated telemetry reader: the remote loopback snapshot may take
		// 220ms; it must NEVER block 40ms keyboard/joystick polling.
		telemetrySnapshots := make(chan telemetryResult, 1)
		go func() {
			clock := time.NewTicker(350 * time.Millisecond)
			defer clock.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-clock.C:
					v := s.telemetry.read()
					select {
					case telemetrySnapshots <- v:
					default:
						select { case <-telemetrySnapshots: default: }
						select { case telemetrySnapshots <- v: default: }
					}
				}
			}
		}()
		var observedTelemetry telemetryResult
		var lastTelemetryAt time.Time
		var lastDLLProbe physicalProbeReport
		var lastDLLProbeCheck time.Time
		var plannedAngle float64
		var plannedAvailable bool
		var lastPlanTick time.Time
		var physicalSelectionSignature string
		var profiles runtimeProfiles
		var lastProfileRead time.Time
		var lastInput string
		var lastEventTime time.Time
		var manual, prevToggle, prevLeft, prevRight, prevAnalog bool
		var prevApprovedDirection int
		var profileID string
		var profileLoaded bool
		var lastModeRequest uint64
		var gameRunning bool
		var activationCount, leftCount, rightCount uint64
		var allowedLeftCount, allowedRightCount uint64
		var analogLeftCount, analogRightCount uint64
		var lastApprovedCommand string
		var lastApprovedTime time.Time
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
				if diagnostic != nil && diagnostic.gameWasRunning {
					diagnostic.report.LastGameStopped=time.Now().UTC().Format(time.RFC3339)
					diagnostic.flush(time.Now())
				}
				publishInputLinkSnapshot(inputLinkSnapshot{})
				return
			case <-ticker.C:
			}
			now := time.Now()
			if now.Sub(lastDLLProbeCheck) >= 500*time.Millisecond {
				lastDLLProbe = readPhysicalDLLProbe()
				lastDLLProbeCheck = now
			}
			select {
			case v := <-telemetrySnapshots:
				observedTelemetry = v
				lastTelemetryAt = now
			default:
			}
			dt := 0.04
			if !lastPlanTick.IsZero() {
				dt = now.Sub(lastPlanTick).Seconds()
			}
			lastPlanTick = now
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
				prevApprovedDirection = 0
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
				allowedLeftCount, allowedRightCount = 0, 0
				analogLeftCount, analogRightCount = 0, 0
				prevApprovedDirection = 0
				lastApprovedCommand = ""
				lastApprovedTime = time.Time{}
				plannedAngle = 0
				plannedAvailable = false
				history = nil
			}
			manual = retainManualOnFocusChange(manual, gameRunning)
			if seq, enabled, updated := readInputModeRequest(lastModeRequest); updated {
				lastModeRequest = seq
				// The UI button is a request for the same TEST mode. It never
				// grants permission to send an enabled physical DLL packet.
				if !enabled || (gameRunning && found) {
					manual = enabled
					activationCount++
					lastInput = "Activation (bouton EssieuPilot)"
					if !manual { lastInput = "Désactivation (bouton EssieuPilot)" }
					lastEventTime = now
					addHistory(lastInput, now)
				}
			}
			focus := detectATSForeground()
			foreground := gameRunning && focus.IsATS
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
				ForegroundPID:          focus.ForegroundPID,
				ATSProcessPID:          focus.ATSProcessPID,
				FocusReason:            focus.Reason,
				LeftVirtualKey:         codeToVK(active.LeftKey),
				RightVirtualKey:        codeToVK(active.RightKey),
				InputMonitoringActive:  foreground && found,
				PhysicalControlEnabled: false,
				PhysicalExperimentOptIn: physicalResearchOptIn,
				PhysicalDLLLoaded: gameRunning && lastDLLProbe.Loaded,
				PhysicalDLLSDKAccepted: gameRunning && lastDLLProbe.Recognized,
				PhysicalDLLGameVersion: lastDLLProbe.GameVersion,
				PhysicalDLLProbeMessage: lastDLLProbe.Reason,
				PhysicalValidationStatus: "Original.12 R&D : le dernier essai EN JEU est enregistré dans Diagnostic ; physique et Convoi non validés.",
				DLLConfirmed:           false,
				StatusMessage: "Original.12 essai : diagnostic physique sauvegardé pendant ATS au premier plan ; aucune direction réelle garantie.",
			}
			if !foreground {
				// ATS lost focus: do not read input from another application.
				// Keep the last armed/unarmed state for the Diagnostic page. Old code
				// reset manual=false on every Alt+Tab, hiding successful activation.
				prevToggle = false
				prevLeft = false
				prevRight = false
				prevAnalog = false
				prevApprovedDirection = 0
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
					lastInput = "Touche gauche détectée (" + active.LeftKey + ")"
					lastEventTime = now
					addHistory(lastInput, now)
				}
				if risingEdge(right, prevRight) {
					rightCount++
					lastInput = "Touche droite détectée (" + active.RightKey + ")"
					lastEventTime = now
					addHistory(lastInput, now)
				}
				prevLeft = left
				prevRight = right
				analogValue := analogFromDevices(active, devices)
				analogActive := math.Abs(analogValue) > 0.000001
				if risingEdge(analogActive, prevAnalog) {
					lastInput = "Axe analogique détecté (" + active.AnalogAxis + ")"
					lastEventTime = now
					addHistory(lastInput, now)
				}
				prevAnalog = analogActive
				allowedNow := steeringInstructionAllowed(manual, gameRunning, foreground, found)
				command := resolveInputIntent(allowedNow, left, right, analogValue)
				nextDirection, acceptedEdge := authorizedDirectionEdge(prevApprovedDirection, command)
				if acceptedEdge {
					if nextDirection < 0 { allowedLeftCount++ } else { allowedRightCount++ }
					if command.Source == "Joystick analogique" {
						if nextDirection < 0 { analogLeftCount++ } else { analogRightCount++ }
					}
					lastApprovedCommand = command.Direction + " (" + command.Source + ")"
					lastApprovedTime = now
					lastDirection = lastApprovedCommand
					lastDirectionTime = now
					addHistory("Commande autorisée : "+lastApprovedCommand, now)
				}
				prevApprovedDirection = nextDirection
				snapshot.HeldDirection = command.Direction
				snapshot.RequestedSource = command.Source
				snapshot.RequestedPercent = math.Round(command.Normalized*1000)/10
				snapshot.AnalogDetectedPercent = math.Round(analogValue*1000)/10
				if allowedNow && !left && !right {
					snapshot.AnalogAuthorizedPercent = math.Round(analogValue*1000)/10
				}
				snapshot.LeftPressed = command.Direction == "GAUCHE"
				snapshot.RightPressed = command.Direction == "DROITE"
				detectedPreviewID:=""
				if observedTelemetry.Connected && !observedTelemetry.Paused &&
					now.Sub(lastTelemetryAt)<750*time.Millisecond &&
					len(observedTelemetry.Trailers)==1 {
					detectedPreviewID=observedTelemetry.Trailers[0].ID
				}
				selectedPreview, previewMode, selectedForPreview := selectedPhysicalTrailerForATS(active,detectedPreviewID)
				selectionSignature:=active.ID+"|"+detectedPreviewID+"|"+selectedPreview.ID
				if selectionSignature!=physicalSelectionSignature {
					physicalSelectionSignature=selectionSignature
					plannedAngle=0
					plannedAvailable=false
				}
				if !selectedForPreview {
					snapshot.PlannedAngleReason = "Association remorque : "+previewMode+" (Diagnostic → choisir une configuration)."
				} else if axle, compatible := trailerCompatible(selectedPreview); compatible {
					speedValid := observedTelemetry.Connected && !observedTelemetry.Paused &&
						now.Sub(lastTelemetryAt) < time.Second
					speed := 0.0
					if speedValid {speed = observedTelemetry.SpeedKMH}
					// Safe in-memory preview: calibrated physical game output is
					// explicitly blocked until ATS 1.62 DLL is proven.
					plannedAngle = planTestSteeringAngle(plannedAngle, command, dt, speed,
						axle, active.AutoCenter)
					snapshot.PlannedAngleDegrees = math.Round(plannedAngle*100)/100
					snapshot.PlannedAngleValid = true
					plannedAvailable = true
					snapshot.ObservedSpeedKMH = speed
					snapshot.ObservedSpeedValid = speedValid
					if !speedValid {
						snapshot.PlannedAngleReason = "Consigne calculée hors jeu ; vitesse ATS non confirmée : aucun recentrage de simulation"
					} else {
						snapshot.PlannedAngleReason = "Consigne calculée hors jeu ; vitesse ATS vérifiée (télémétrie SCS)"
					}
				} else {
					snapshot.PlannedAngleReason = "Groupes d'essieux incompatibles avec une commande physique unique par remorque"
				}
			}
			if !foreground && plannedAvailable && gameRunning && found {
				// Show last calculated angle on returning to Diagnostic.
				// Not a live wheel-angle measurement; input is paused.
				snapshot.PlannedAngleDegrees = math.Round(plannedAngle*100)/100
				snapshot.PlannedAngleValid = true
				snapshot.PlannedAngleReason = "Dernière consigne calculée en jeu (hors physique) ; entrée suspendue"
			}
			snapshot.ManualArmed = manual
			snapshot.SteeringInputAllowed = steeringInstructionAllowed(manual, gameRunning, foreground, found)
			if !gameRunning {
				snapshot.InputReason = "ATS n'est pas lancé"
			} else if !found {
				snapshot.InputReason = "Profil actif introuvable ou non enregistré"
			} else if !foreground {
				snapshot.InputReason = focus.Reason + " : touches suspendues hors du jeu"
			} else {
				snapshot.InputReason = "Surveillance clavier/manette active dans ATS"
				if strings.HasPrefix(active.LeftKey, "KEY:") && snapshot.LeftVirtualKey == 0 {
					snapshot.InputReason = "Touche gauche non reconnue : vérifier la configuration"
				}
				if strings.HasPrefix(active.RightKey, "KEY:") && snapshot.RightVirtualKey == 0 {
					snapshot.InputReason = "Touche droite non reconnue : vérifier la configuration"
				}
			}
			snapshot.ActivationCount = activationCount
			snapshot.LeftCount = leftCount
			snapshot.RightCount = rightCount
			snapshot.AllowedLeftCount = allowedLeftCount
			snapshot.AllowedRightCount = allowedRightCount
			snapshot.AnalogLeftCount = analogLeftCount
			snapshot.AnalogRightCount = analogRightCount
			snapshot.LastAuthorizedCommand, snapshot.LastAuthorizedTime = formatInputEvent(lastApprovedCommand,lastApprovedTime)
			snapshot.LastDirection = lastDirection
			_, snapshot.LastDirectionTime = formatInputEvent(lastDirection, lastDirectionTime)
			if !lastFocusTime.IsZero() {snapshot.LastFocusTime = lastFocusTime.Format(time.RFC3339)}
			snapshot.InputHistory = append([]string(nil), history...)
			snapshot.LastInput, snapshot.LastEventTime = formatInputEvent(lastInput, lastEventTime)
			if gameRunning && observedTelemetry.Connected && observedTelemetry.WheelSteeringValid &&
				now.Sub(lastTelemetryAt) < time.Second {
				snapshot.ObservedWheelSteeringValid = true
				snapshot.ObservedWheelSteeringDegrees = append([]float64(nil), observedTelemetry.WheelSteeringDegrees...)
			}
			// R&D ONLY. All input types share the same ACTIF/INACTIF gate,
			// plus explicit unsafe experiment consent, a 1.62 DLL SDK probe,
			// fresh attached-trailer telemetry, and compatible axle settings.
			// ATS offsets are still unverified; do not ship this enabled path.
			emitted := false
			speedAndTrailerValid := observedTelemetry.Connected && !observedTelemetry.Paused &&
				now.Sub(lastTelemetryAt) < 750*time.Millisecond
			oneTrailer := speedAndTrailerValid && len(observedTelemetry.Trailers) == 1
			var physicalAxle runtimeAxle
			detectedID:=""
			if oneTrailer { detectedID=observedTelemetry.Trailers[0].ID }
			physicalTrailer, associationMode, selected := selectedPhysicalTrailerForATS(active,detectedID)
			// The explicit per-preset ATS binding is more specific than a legacy
			// profile-level match_key. A profile can now store arbitrary trailers.
			profileMatched := oneTrailer && (
				associationMode=="ASSOCIATION_AUTOMATIQUE_EXACTE" ||
				active.MatchKey=="" || strings.EqualFold(active.MatchKey,detectedID))
			snapshot.PhysicalSelectedTrailerID=physicalTrailer.ID
			snapshot.PhysicalDetectedTrailerID=detectedID
			snapshot.PhysicalSelectionReason=associationMode
			axlesCompatible := selected
			if axlesCompatible {
				physicalAxle, axlesCompatible = trailerCompatible(physicalTrailer)
			}
			conditions := experimentalPhysicalConditions{
				ExplicitResearchOptIn: physicalResearchOptIn,
				InputAuthorized: steeringInstructionAllowed(manual, gameRunning, foreground, found),
				DLLSDKProbeRecognized: lastDLLProbe.Recognized && lastDLLProbe.Loaded,
				FreshSCSTelemetry: speedAndTrailerValid,
				OneAttachedTrailer: oneTrailer,
				ActiveProfileMatched: profileMatched,
				CompatibleAxles: axlesCompatible,
			}
			allowed := allowExperimentalPhysics(conditions)
			reasons := physicalGateReasons(conditions,bridge != nil,snapshot.PlannedAngleValid)
			if !selected {reasons=append(reasons,associationMode)}
			if selected && !axlesCompatible {reasons=append(reasons,"ESSIEUX_AVEC_ANGLES_SENS_OU_RATIOS_DIFFERENTS")}
			if bridge != nil && allowed && snapshot.PlannedAngleValid {
				if normalized, ok := normalizedPhysicalAngle(plannedAngle, physicalAxle.MaxAngle, physicalTrailer.PhysicalFullScaleDegrees); ok {
					if err := bridge.publish(true, 0, normalized); err == nil {
						emitted = true
						snapshot.PhysicalPacketSent = true
						snapshot.PhysicalPacketNormalized = float64(normalized)
						snapshot.PhysicalControlEnabled = true // IPC request, NOT wheel proof.
					snapshot.PhysicalCalibrationReferenceDegrees=physicalReferenceDegrees(physicalTrailer.PhysicalFullScaleDegrees)
						ack := readPhysicalAck()
						snapshot.PhysicalCallbackAck = ack.Received && ack.TrailerIndex == 0
						if snapshot.PhysicalCallbackAck {
							snapshot.PhysicalCallbackAckAgeMs = ack.AgeMs
						}
						snapshot.PhysicalValidationStatus = "R&D : paquet IPC envoyé ; callback DLL non prouvé ; roues physiques NON VALIDÉES"
						if snapshot.PhysicalCallbackAck {
							snapshot.PhysicalValidationStatus = "R&D : fonction de braquage appelée dans DLL ; visuel et physique NON VALIDÉS"
						}
					} else {
						snapshot.StatusMessage = "Échec d'envoi physique expérimental : " + err.Error()
						reasons=append(reasons,"ERREUR_ECRITURE_IPC")
					}
				} else {
					reasons=append(reasons,"ANGLE_PHYSIQUE_NON_NORMALISABLE")
				}
			}
			if bridge != nil && !emitted {
				// Cancel and restore ATS default steering on any failure,
				// inactive input, stale telemetry, focus loss or closed game.
				if err := bridge.publish(false, 0, 0); err != nil {
					snapshot.IPCReady = false
					snapshot.StatusMessage = "Canal IPC indisponible : " + err.Error()
				}
			}
			// Persist the reason while ATS has focus, before Alt+Tab or game exit.
			// Recorded ACK is callback evidence only; never imply wheel motion.
			if diagnostic != nil {
				diagnostic.observe(now,snapshot,conditions,reasons,len(observedTelemetry.Trailers))
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

// Steering direction is a command only when the user has explicitly armed
// the mode, ATS is running AND foreground, and the saved profile is valid.
// This gate also applies to future physical output; stage 1 still publishes
// exclusively neutral packets to the prototype DLL.
func steeringInstructionAllowed(armed, gameRunning, foreground, profileFound bool) bool {
	return armed && gameRunning && foreground && profileFound
}

// Each controller tick evaluates held button state rather than reusing
// rising-edge events. One held press is sufficient; contradictory inputs
// cancel each other. This is a DIAGNOSTIC only until the ATS DLL is validated.
func heldDirection(authorized, left, right bool) string {
	if !authorized || left == right { return "" }
	if left {return "GAUCHE"}
	return "DROITE"
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
