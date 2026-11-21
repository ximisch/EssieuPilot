//go:build windows

package main

import (
 "math"
 "testing"
)

// A signed left/right request must remain signed through the Go steering
// planner and the normalized IPC encoder. This does not prove ATS wheel motion.
func TestOriginal20SymmetricDigitalSteeringAndNormalizedIPCAngles(t *testing.T) {
 axle:=runtimeAxle{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}
 left:=resolveInputIntent(true,true,false,0)
 right:=resolveInputIntent(true,false,true,0)
 if left.Normalized!=-1 || right.Normalized!=1 {t.Fatal("key direction mapping reversed")}
 for _,tc:=range []struct{name string;command inputIntent;sign float64}{
  {"gauche",left,-1},{"droite",right,1},
 } {
  t.Run(tc.name,func(t *testing.T){
   angle:=0.0
   for i:=0;i<100;i++{
    angle=planTestSteeringAngle(angle,tc.command,.04,0,axle,true)
   }
   if math.Abs(angle-tc.sign*35)>1e-7 {t.Fatalf("angle=%v, expected=%v",angle,tc.sign*35)}
   normalized,ok:=normalizedPhysicalAngle(angle,35,60)
   if !ok || math.Abs(float64(normalized)-tc.sign*35.0/60.0)>0.00001 {
    t.Fatalf("signed IPC angle=%v valid=%v",normalized,ok)
   }
   data:=make([]byte,steeringPacketSize)
   if err:=encodeSteeringPacket(data,steeringPacket{Seq:2,Enabled:true,UptimeMs:1500,TrailerIndex:0,NormalizedAngle:normalized});err!=nil{t.Fatal(err)}
   decoded,err:=decodeSteeringPacket(data,1550)
   if err!=nil || decoded.NormalizedAngle!=normalized {t.Fatalf("wire IPC lost sign: %+v %v",decoded,err)}
  })
 }
}

// A saved per-trailer reversal is a user preference, never an automatically
// guessed correction to a one-sided physical DLL.
func TestOriginal20SavedReversalChangesSignWithoutChangingLimit(t *testing.T) {
 axle:=runtimeAxle{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1,Reversed:true}
 got:=planTestSteeringAngle(0,resolveInputIntent(true,true,false,0),.1,0,axle,true)
 if got<=0 || got>35 {t.Fatalf("reversed left must be positive but bounded, got %v",got)}
 other:=planTestSteeringAngle(0,resolveInputIntent(true,false,true,0),.1,0,axle,true)
 if other>=0 || other<(-35) {t.Fatalf("reversed right must be negative but bounded, got %v",other)}
}
