//go:build windows
package main

import ("math";"testing")

func ptrDegrees(v float64)*float64{return &v}

func TestOriginal15PhysicalCapIsIndependentFromSavedAxle(t *testing.T) {
    saved:=runtimeAxle{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}
    trailer:=runtimeTrailer{ID:"lowboy",PhysicalAngleLimitDegrees:ptrDegrees(0)}
    cap,ok:=effectivePhysicalAxle(trailer,saved)
    if !ok || cap.MaxAngle!=0 || saved.MaxAngle!=35 {t.Fatal("zero cap must preserve original 35-degree axle configuration")}
    normalized,valid:=normalizedPhysicalAngle(0,cap.MaxAngle,60)
    if !valid || normalized!=0 {t.Fatalf("zero cap must send enabled zero IPC target: %v %v",normalized,valid)}
    cap2,ok:=effectivePhysicalAxle(runtimeTrailer{PhysicalAngleLimitDegrees:ptrDegrees(15)},saved)
    if !ok||cap2.MaxAngle!=15 {t.Fatal("15 degree physical cap not honored")}
    normalized,valid=normalizedPhysicalAngle(15,cap2.MaxAngle,60)
    if !valid||math.Abs(float64(normalized)-.25)>1e-6 {t.Fatal("15 degree physical cap not proportional")}
    defaultCap,ok:=effectivePhysicalAxle(runtimeTrailer{},saved)
    if !ok||defaultCap.MaxAngle!=35 {t.Fatal("legacy physical cap must use stored axle maximum")}
    for _,invalid:=range []float64{-1,35.5,61,math.NaN(),math.Inf(1)} {
        if _,ok:=effectivePhysicalAxle(runtimeTrailer{PhysicalAngleLimitDegrees:ptrDegrees(invalid)},saved);ok {
            t.Fatalf("invalid cap accepted: %v",invalid)
        }
    }
}
func TestOriginal15SavedZeroAxleSettingsAllowStraightIPC(t *testing.T) {
    tr:=runtimeTrailer{ID:"new",Axles:[]runtimeAxle{
        {Enabled:true,MaxAngle:0,TurnRate:14,CenterRate:7,Ratio:1},
        {Enabled:true,MaxAngle:0,TurnRate:14,CenterRate:7,Ratio:1},
    }}
    a,ok:=trailerCompatible(tr)
    if !ok || a.MaxAngle!=0 {t.Fatalf("zero-degree axles unexpectedly rejected: %+v %v",a,ok)}
    cmd:=resolveInputIntent(true,false,true,0)
    target:=planTestSteeringAngle(12,cmd,.04,0,a,true)
    if target!=0 {t.Fatalf("saved 0-degree axle must center the physical target: %v",target)}
    if n,ok:=normalizedPhysicalAngle(target,a.MaxAngle,60);!ok||n!=0 {t.Fatal("zero target must be representable by IPC")}
}
