package main

import "math"

// planTestSteeringAngle calculates an *unapplied*, in-memory steering target
// for an individual compatible axle. It mirrors the original 2D/3D behavior
// and will only be connected to a real ATS hook after verifying the game's
// steering layout, wheel units and in-game controls. The absence of valid
// telemetry never authorizes a real output.
func planTestSteeringAngle(before float64, cmd inputIntent, dt, speedKMH float64, axle runtimeAxle, autoCenter bool) float64 {
	if !isFinite(before) || !isFinite(dt) || !isFinite(speedKMH) ||
		!isFinite(axle.MaxAngle) || !isFinite(axle.Ratio) ||
		!isFinite(axle.TurnRate) || !isFinite(axle.CenterRate) {
		return 0
	}
	if dt < 0 || axle.MaxAngle < 0 || axle.MaxAngle > 60 || axle.Ratio <= 0 || axle.Ratio > 2 ||
		axle.TurnRate < 0 || axle.TurnRate > 90 || axle.CenterRate < 0 || axle.CenterRate > 90 {
		return 0
	}
	limit := axle.MaxAngle
	before = math.Max(-limit, math.Min(limit, before))
	dt = math.Min(dt, 0.1)
	if !axle.Enabled {
		return before
	}
	sign := 1.0
	if axle.Reversed {
		sign = -1
	}
	if cmd.Source == "Joystick analogique" && math.Abs(cmd.Normalized) > 0.000001 {
		// Stick position, NOT an angular velocity: holding 30% of 35 degrees
		// requests 10.5 degrees; returning to zero holds at standstill.
		target := cmd.Normalized * limit * axle.Ratio * sign
		return math.Max(-limit, math.Min(limit, target))
	}
	if cmd.Source == "Touche / bouton" && math.Abs(cmd.Normalized) > 0.000001 {
		target := math.Copysign(limit, cmd.Normalized*sign)
		step := axle.TurnRate * dt * axle.Ratio
		if target < before {
			return math.Max(target, before-step)
		}
		return math.Min(target, before+step)
	}
	if autoCenter && math.Abs(speedKMH) >= 0.2 {
		step := axle.CenterRate * dt
		if math.Abs(before) <= step {
			return 0
		}
		return before - math.Copysign(step, before)
	}
	return before
}
