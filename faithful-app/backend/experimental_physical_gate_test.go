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
	got,ok:=normalizedPhysicalAngle(10.5,35)
	if !ok||math.Abs(float64(got)-.3)>.000001{
		t.Fatalf("30%% of 35deg should produce 0.3 normalized, got %v %v",got,ok)
	}
	for _,v:=range []struct{angle,max float64}{
		{math.NaN(),35},{0,math.Inf(1)},{36,35},{0,0},{0,70},{math.Inf(-1),35},
	} {
		if _,ok:=normalizedPhysicalAngle(v.angle,v.max);ok {
			t.Fatalf("accepted uncalibrated invalid physical angle %v/%v",v.angle,v.max)
		}
	}
}
