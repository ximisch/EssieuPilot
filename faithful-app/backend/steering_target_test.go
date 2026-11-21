package main

import (
	"math"
	"testing"
)

func TestSteeringTargetMatchesOriginalSimulatorAtStandstill(t *testing.T) {
	axle := runtimeAxle{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}
	analog := resolveInputIntent(true,false,false,.30)
	got := planTestSteeringAngle(0,analog,.04,0,axle,true)
	if math.Abs(got-10.5)>1e-8 { t.Fatalf("30 percent stick should request 10.5 degrees, got %v",got) }
	got = planTestSteeringAngle(got,inputIntent{},.04,0,axle,true)
	if math.Abs(got-10.5)>1e-8 {t.Fatalf("neutral stick at standstill should hold, got %v",got)}
	got = planTestSteeringAngle(got,inputIntent{},.1,10,axle,true)
	if math.Abs(got-9.8)>1e-8 {t.Fatalf("moving recenter 7deg/s expected 9.8, got %v",got)}
	got = planTestSteeringAngle(got,resolveInputIntent(false,false,false,.75),.1,0,axle,true)
	if math.Abs(got-9.8)>1e-8 {t.Fatalf("disabled analog changed target: %v",got)}
}

func TestSteeringTargetHeldDigitalLimitAndReversal(t *testing.T) {
	axle := runtimeAxle{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}
	a := 0.0
	for i:=0;i<100;i++{
		a=planTestSteeringAngle(a,resolveInputIntent(true,false,true,0),.04,0,axle,true)
	}
	if math.Abs(a-35)>1e-8 {t.Fatalf("held right should stop at 35, got %v",a)}
	for i:=0;i<100;i++{
		a=planTestSteeringAngle(a,resolveInputIntent(true,true,false,0),.04,0,axle,true)
	}
	if math.Abs(a-(-21))>1e-8 {t.Fatalf("held left should continuously turn through zero at 14deg/s, got %v",a)}
	axle.Reversed=true
	a=planTestSteeringAngle(0,resolveInputIntent(true,false,false,.3),.04,0,axle,true)
	if math.Abs(a-(-10.5))>1e-8 {t.Fatalf("reversed axle sign ignored, got %v",a)}
}

func TestSteeringTargetRejectsInvalidParameters(t *testing.T) {
	axle:=runtimeAxle{Enabled:true,MaxAngle:100,TurnRate:14,CenterRate:7,Ratio:1}
	if angle:=planTestSteeringAngle(5,resolveInputIntent(true,false,true,0),.04,0,axle,true);angle!=0 {
		t.Fatalf("unbounded physical angle accepted %v",angle)
	}
	axle.MaxAngle=35
	if angle:=planTestSteeringAngle(5,inputIntent{},math.NaN(),0,axle,true);angle!=0 {
		t.Fatal("NaN time delta was accepted")
	}
}
