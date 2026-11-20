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
