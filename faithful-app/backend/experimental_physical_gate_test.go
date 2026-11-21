package main

import (
	"math"
	"testing"
)

func TestPhysicalGateRequiresEveryCondition(t *testing.T) {
	full:=experimentalPhysicalConditions{
		ExplicitResearchOptIn:true,
		InputAuthorized:true,
		DLLSDKProbeRecognized:true,
		FreshSCSTelemetry:true,
		OneAttachedTrailer:true,
		ActiveProfileMatched:true,
		CompatibleAxles:true,
	}
	if !allowExperimentalPhysics(full) {
		t.Fatal("fully authorized opt-in should reach experimental command path")
	}
	tests:=[]struct{name string; edit func(*experimentalPhysicalConditions)}{
		{"no explicit consent",func(c *experimentalPhysicalConditions){c.ExplicitResearchOptIn=false}},
		{"pilotage inactive",func(c *experimentalPhysicalConditions){c.InputAuthorized=false}},
		{"dll missing or wrong ATS",func(c *experimentalPhysicalConditions){c.DLLSDKProbeRecognized=false}},
		{"telemetry stale or paused",func(c *experimentalPhysicalConditions){c.FreshSCSTelemetry=false}},
		{"not exactly one trailer",func(c *experimentalPhysicalConditions){c.OneAttachedTrailer=false}},
		{"wrong saved profile",func(c *experimentalPhysicalConditions){c.ActiveProfileMatched=false}},
		{"conflicting axle settings",func(c *experimentalPhysicalConditions){c.CompatibleAxles=false}},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T){
			c:=full;tt.edit(&c)
			if allowExperimentalPhysics(c) {t.Fatal("physical steering was allowed with failed safety condition")}
		})
	}
}

func TestNormalizePhysicalAngleRND(t *testing.T) {
	got,ok:=normalizedPhysicalAngle(10.5,35,60)
	if !ok||math.Abs(float64(got)-.175)>.000001{
		t.Fatalf("10.5 nominal degrees at 60 degree calibration should produce 0.175 normalized, got %v %v",got,ok)
	}
	for _,v:=range []struct{angle,max float64}{
		{math.NaN(),35},{0,math.Inf(1)},{36,35},{0,-1},{0,70},{math.Inf(-1),35},
	} {
		if _,ok:=normalizedPhysicalAngle(v.angle,v.max,60);ok {
			t.Fatalf("accepted uncalibrated invalid physical angle %v/%v",v.angle,v.max)
		}
	}
}

func TestPhysicalCalibrationDecouplesPresetLimitFromATSFullLock(t *testing.T) {
    got15,ok15:=normalizedPhysicalAngle(15,15,60)
    got35,ok35:=normalizedPhysicalAngle(35,35,60)
    if !ok15||!ok35||math.Abs(float64(got15)-.25)>1e-6||
        math.Abs(float64(got35)-35.0/60)>1e-6 || got15>=got35 {
        t.Fatalf("original13 full-lock regression: 15=%v %t, 35=%v %t",got15,ok15,got35,ok35)
    }
    if _,ok:=normalizedPhysicalAngle(35,35,30);ok {t.Fatal("invalid reference under 35 accepted")}
    if _,ok:=normalizedPhysicalAngle(45,45,35);ok {t.Fatal("reference smaller than configured physical limit accepted")}
    if x,ok:=normalizedPhysicalAngle(-35,35,120);!ok||x>=0 {t.Fatal("reverse steering not preserved")}
    if got:=physicalReferenceDegrees(0);got!=60 {t.Fatal("legacy preset should use 60 reference")}
    for _,v:=range []float64{-1,34,121,math.NaN(),math.Inf(1)} {
        if physicalReferenceDegrees(v)!=0 {t.Fatalf("invalid reference %v accepted",v)}
    }
}

func TestOriginal15PhysicalZeroIsStraight(t *testing.T) {
    for _,ref:=range []float64{0,35,60,120} {
        v,ok:=normalizedPhysicalAngle(0,0,ref)
        if !ok || v!=0 {t.Fatalf("zero steering rejected: %v %v ref=%v",v,ok,ref)}
    }
    if _,ok:=normalizedPhysicalAngle(1,0,60);ok {t.Fatal("nonzero target when cap=0 accepted")}
    if _,ok:=normalizedPhysicalAngle(-.1,0,60);ok {t.Fatal("negative steering when cap=0 accepted")}
}
