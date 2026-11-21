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

// ATS accepts a normalized [-1,+1] steering command, NOT degrees.
// Original.13 divided by each preset's maxAngle, so any configured maximum
// (15 / 15, 35 / 35, ...) became full lock. Original.14 uses a separate
// calibration reference, default 60 nominal degrees at full lock.
// IMPORTANT: the telemetry wheel-steering values were zero in the user's
// test; this is an adjustable approximation, not a measured wheel angle.
func physicalReferenceDegrees(configured float64) float64 {
    if configured==0 {return 60}
    if math.IsNaN(configured)||math.IsInf(configured,0) ||
        configured < 35 || configured > 120 {return 0}
    return configured
}

func normalizedPhysicalAngle(angle, maxAngle, physicalFullScale float64) (float32, bool) {
    ref:=physicalReferenceDegrees(physicalFullScale)
    if math.IsNaN(angle)||math.IsInf(angle,0)||
        math.IsNaN(maxAngle)||math.IsInf(maxAngle,0)||
        maxAngle<=0||maxAngle>60||ref==0||ref<maxAngle||
        math.Abs(angle)>maxAngle+0.0001 {
        return 0,false
    }
    return float32(math.Max(-1,math.Min(1,angle/ref))),true
}
