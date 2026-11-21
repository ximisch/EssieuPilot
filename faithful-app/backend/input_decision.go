package main

import "math"

// inputIntent is the effective direction request from the user's saved
// keyboard, joystick-button or proportional analog bindings.
// It is a *diagnostic request*: physical ATS output is disabled until the
// game-version-specific steering hook is verified in a controlled test.
type inputIntent struct {
	Direction  string
	Source     string
	Normalized float64
}

// A digital direction takes precedence over analog, as in the original
// 2D/3D simulator. Opposing digital buttons cancel each other.
// All three input types pass through the same manual-mode safety gate.
func resolveInputIntent(authorized, left, right bool, analog float64) inputIntent {
	if !authorized {
		return inputIntent{}
	}
	if left && right {
		return inputIntent{Source: "Touches opposées — annulées"}
	}
	if left {
		return inputIntent{Direction: "GAUCHE", Source: "Touche / bouton", Normalized: -1}
	}
	if right {
		return inputIntent{Direction: "DROITE", Source: "Touche / bouton", Normalized: 1}
	}
	if math.IsNaN(analog) || math.IsInf(analog, 0) {
		return inputIntent{}
	}
	analog = math.Max(-1, math.Min(1, analog))
	if math.Abs(analog) <= 0.000001 {
		return inputIntent{}
	}
	if analog < 0 {
		return inputIntent{Direction: "GAUCHE", Source: "Joystick analogique", Normalized: analog}
	}
	return inputIntent{Direction: "DROITE", Source: "Joystick analogique", Normalized: analog}
}

// Count transitions rather than polling ticks: a held input is one command,
// not 25 repeated commands per second. Inactive input never increments.
func authorizedDirectionEdge(previous int, command inputIntent) (int, bool) {
	current := 0
	switch command.Direction {
	case "GAUCHE":
		current = -1
	case "DROITE":
		current = 1
	}
	return current, current != 0 && current != previous
}
