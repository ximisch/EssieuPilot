package main

import (
	"math"
	"testing"
)

func TestInputAuthorizationAllDevices(t *testing.T) {
	testCases := []struct {
		name string
		authorized, left, right bool
		analog float64
		want inputIntent
	}{
		{"inactive keyboard", false, true, false, 0, inputIntent{}},
		{"inactive joystick button", false, false, true, 0, inputIntent{}},
		{"inactive joystick analog", false, false, false, .30, inputIntent{}},
		{"active left key", true, true, false, 0, inputIntent{"GAUCHE", "Touche / bouton", -1}},
		{"active right button", true, false, true, 0, inputIntent{"DROITE", "Touche / bouton", 1}},
		{"active analog 30 percent", true, false, false, .30, inputIntent{"DROITE", "Joystick analogique", .30}},
		{"active analog reverse", true, false, false, -.55, inputIntent{"GAUCHE", "Joystick analogique", -.55}},
		{"digital priority", true, true, false, .85, inputIntent{"GAUCHE", "Touche / bouton", -1}},
		{"opposing keys neutral", true, true, true, .5, inputIntent{"", "Touches opposées — annulées", 0}},
		{"neutral joystick", true, false, false, 0, inputIntent{}},
		{"non-finite joystick", true, false, false, math.NaN(), inputIntent{}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveInputIntent(tc.authorized, tc.left, tc.right, tc.analog)
			if got != tc.want {
				t.Fatalf("got %+v; expected %+v", got, tc.want)
			}
		})
	}
}

func TestHeldCommandsCountOnceAndResetOnRelease(t *testing.T) {
	last := 0
	for tick := 0; tick < 75; tick++ {
		intent := resolveInputIntent(true, false, true, 0)
		var edge bool
		last, edge = authorizedDirectionEdge(last, intent)
		if edge != (tick == 0) {
			t.Fatalf("held key counted again at tick %d", tick)
		}
	}
	last, _ = authorizedDirectionEdge(last, resolveInputIntent(true, false, false, 0))
	if last != 0 {
		t.Fatal("release did not reset direction")
	}
	last, edge := authorizedDirectionEdge(last, resolveInputIntent(true, false, true, 0))
	if last != 1 || !edge {
		t.Fatal("new keypress after release not recognized")
	}
	last, edge = authorizedDirectionEdge(last, resolveInputIntent(false, false, true, .6))
	if last != 0 || edge {
		t.Fatal("disabling pilotage failed to clear input")
	}
	last, edge = authorizedDirectionEdge(last, resolveInputIntent(true, false, false, -.3))
	if last != -1 || !edge {
		t.Fatal("analog command not registered after reactivation")
	}
	last, edge = authorizedDirectionEdge(last, resolveInputIntent(true, false, false, -.8))
	if last != -1 || edge {
		t.Fatal("analog held on same side should not count multiple times")
	}
	last, edge = authorizedDirectionEdge(last, resolveInputIntent(true, false, false, +.25))
	if last != 1 || !edge {
		t.Fatal("crossing analog sides should count as a new direction")
	}
}
