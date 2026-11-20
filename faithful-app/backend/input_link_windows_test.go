//go:build windows

package main

import "testing"

func TestKeysAreResolvedFromChangedSavedBindings(t *testing.T) {
	if codeToVK("KEY:F8") != 0x77 || codeToVK("KEY:F9") != 0x78 ||
		codeToVK("KEY:F10") != 0x79 {
		t.Fatal("default F8 F9 F10 mapping invalid")
	}
	if codeToVK("KEY:F12") != 0x7B || codeToVK("KEY:KeyJ") != 'J' ||
		codeToVK("KEY:ArrowLeft") != 0x25 || codeToVK("KEY:Digit4") != '4' {
		t.Fatal("custom keyboard binding ignored")
	}
	if codeToVK("KEY:NotAKey") != 0 {
		t.Fatal("untrusted custom binding accepted")
	}
}

func TestGamepadAndJoystickBindingsFollowSavedProfile(t *testing.T) {
	devices := []device{{ID: "JOY2", Buttons: []bool{false, true, false, true}, Axes: []float64{-0.84, 0.7}, POV: 3}}
	if !deviceBindingDown("DEVICE:JOY2:BUTTON:3", devices) ||
		deviceBindingDown("DEVICE:JOY2:BUTTON:2", devices) ||
		!deviceBindingDown("DEVICE:JOY2:AXIS:0:-", devices) ||
		!deviceBindingDown("DEVICE:JOY2:POV:3", devices) {
		t.Fatal("changed joystick input mapping ignored")
	}
}

func TestOnlySavedActiveProfileIsUsed(t *testing.T) {
	profiles := runtimeProfiles{ActiveID: "custom", Profiles: []runtimeProfile{
		{ID: "default", Name: "Default", ToggleKey: "KEY:F8"},
		{ID: "custom", Name: "My remorque", ToggleKey: "KEY:F12", LeftKey: "KEY:KeyJ", RightKey: "KEY:KeyK"},
	}}
	active, ok := activeSavedProfile(profiles)
	if !ok || active.Name != "My remorque" || active.ToggleKey != "KEY:F12" ||
		active.LeftKey != "KEY:KeyJ" || active.RightKey != "KEY:KeyK" {
		t.Fatalf("profile binding changes ignored: %+v", active)
	}
	profiles.ActiveID = "unknown"
	if _, ok = activeSavedProfile(profiles); ok {
		t.Fatal("unknown active profile should not issue any control input")
	}
}

func TestInputEdgesAreRecognizedWhenNotArmed(t *testing.T) {
    // The diagnostic must record raw input even before manual steering is enabled.
    if !risingEdge(true, false) {
        t.Fatal("freshly pressed left/right key not detected")
    }
    if risingEdge(true, true) || risingEdge(false, true) {
        t.Fatal("held or released key incorrectly counted as a new press")
    }
}

func TestManualStateSurvivesReturningToDiagnostic(t *testing.T) {
    // The previous release erased the armed state on every Alt+Tab and hid it
    // when the user returned to EssieuPilot to examine the diagnostic.
    if !retainManualOnFocusChange(true, true) {
        t.Fatal("switching windows must not reset activation while ATS runs")
    }
    if retainManualOnFocusChange(true, false) {
        t.Fatal("closing ATS must disarm manual control")
    }
    if retainManualOnFocusChange(false, true) {
        t.Fatal("switching windows must not turn on disabled steering")
    }
}

func TestForegroundWindowRecognizesGamePIDWithoutOpenProcess(t *testing.T) {
    if !foregroundMatchesATS(420, []uint32{412,420,600}) {
        t.Fatal("ATS foreground PID missed among multiple running processes")
    }
    if foregroundMatchesATS(888, []uint32{412,420,600}) {
        t.Fatal("capturing keys while unrelated process is foreground")
    }
    if foregroundMatchesATS(0, []uint32{0,420}) {
        t.Fatal("invalid Windows foreground PID must never allow capture")
    }
    if foregroundMatchesATS(420, nil) {
        t.Fatal("missing ATS process must never allow capture")
    }
}
