package main

import "math"

// This decision is deliberately distinct from ACTIF/INACTIF. A successful
// keyboard/joystick diagnostic NEVER implicitly authorizes memory hooks.
type experimentalPhysicalConditions struct {
	ExplicitResearchOptIn bool
	InputAuthorized       bool
	DLLSDKProbeRecognized bool
	FreshSCSTelemetry     bool
	OneAttachedTrailer    bool
	ActiveProfileMatched  bool
	CompatibleAxles       bool
}

func allowExperimentalPhysics(c experimentalPhysicalConditions) bool {
	return c.ExplicitResearchOptIn && c.InputAuthorized &&
		c.DLLSDKProbeRecognized && c.FreshSCSTelemetry &&
		c.OneAttachedTrailer && c.ActiveProfileMatched && c.CompatibleAxles
}

// The unit conversion from requested degrees to the internal trailer command
// is still EXPERIMENTAL. This ratio must be calibrated against ATS 1.62 before
// shipping; returning false refuses invalid or unsafe values.
func normalizedPhysicalAngle(angle, maxAngle float64) (float32, bool) {
	if math.IsNaN(angle) || math.IsInf(angle,0) ||
		math.IsNaN(maxAngle) || math.IsInf(maxAngle,0) ||
		maxAngle <= 0 || maxAngle > 60 || math.Abs(angle) > maxAngle+0.0001 {
		return 0,false
	}
	return float32(math.Max(-1,math.Min(1,angle/maxAngle))),true
}
