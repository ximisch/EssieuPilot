package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func allPhysicalConditions() experimentalPhysicalConditions {
 return experimentalPhysicalConditions{ExplicitResearchOptIn:true,InputAuthorized:true,
 DLLSDKProbeRecognized:true,FreshSCSTelemetry:true,
 OneAttachedTrailer:true,ActiveProfileMatched:true,CompatibleAxles:true}
}
func TestOriginal15GateReportsEveryBlocker(t *testing.T) {
 reasons:=physicalGateReasons(experimentalPhysicalConditions{},false,false)
 if len(reasons)!=9 {t.Fatalf("want 9 blockers; got %d %v",len(reasons),reasons)}
 if len(physicalGateReasons(allPhysicalConditions(),true,true))!=0 {t.Fatal("all valid should have no blockers")}
 cond:=allPhysicalConditions()
 cond.FreshSCSTelemetry=false
 v:=physicalGateReasons(cond,true,true)
 if len(v)!=1 || v[0]!="TELEMETRIE_ABSENTE_EN_PAUSE_OU_PERIMEE" {t.Fatalf("unexpected %v",v)}
}

func TestOriginal15PersistsLastFocusedGameResultAfterAltTabAndExit(t *testing.T) {
 dest:=filepath.Join(t.TempDir(),"ATS-Physical-Test","diagnostic-original15.json")
 d:=newPhysicalDiagnosticRecorderAt(dest)
 now:=time.Date(2026,11,21,13,20,0,0,time.UTC)
 c:=allPhysicalConditions()
 active:=inputLinkSnapshot{
  ATSRunning:true, ATSForeground:true, ManualArmed:true,IPCReady:true,
  PhysicalPacketSent:true,PhysicalCallbackAck:false,
  HeldDirection:"DROITE",PlannedAngleDegrees:18.2,
  PhysicalDLLLoaded:true,PhysicalDLLSDKAccepted:true,
 }
 d.observe(now,active,c,nil,1)
 if d.report.LastWithManualInput==nil || d.report.LastInGame==nil || d.report.SentCount!=1 {
  t.Fatalf("missing in-game capture %+v",d.report)
 }
 old:=d.report.LastInGame.At
 // Alt+Tab must not overwrite the actual game sample with 'focus missing'.
 focusLost:=active
 focusLost.ATSForeground=false
 focusLost.PhysicalPacketSent=false
 focusLost.HeldDirection=""
 c.InputAuthorized=false
 d.observe(now.Add(5*time.Second),focusLost,c,physicalGateReasons(c,true,true),1)
 if d.report.LastInGame.At!=old {t.Fatal("Alt+Tab overwrote in-game evidence")}
 focusLost.ATSRunning=false
 d.observe(now.Add(8*time.Second),focusLost,c,nil,0)
 raw,err:=os.ReadFile(dest)
 if err!=nil {t.Fatal(err)}
 var got physicalDiagnosticReport
 if err:=json.Unmarshal(raw,&got);err!=nil{t.Fatal(err)}
 if got.LastWithManualInput==nil || got.LastWithManualInput.RequestedDirection!="DROITE" ||
  !got.LastWithManualInput.IPCSent || got.LastWithManualInput.CallbackACK ||
  got.LastGameStopped=="" || got.SentCount!=1 {t.Fatalf("lost original focused result: %+v",got)}
 if !strings.Contains(got.Warning,"non mesuree") || !strings.Contains(got.Warning,"NON disponible") {
        t.Fatal("missing wheel calibration/independent axle uncertainty notice")
    }
}
func TestOriginal15NeverReportsAckAsPhysicalSteering(t *testing.T) {
 d:=newPhysicalDiagnosticRecorderAt(filepath.Join(t.TempDir(),"report.json"))
 c:=allPhysicalConditions()
 snap:=inputLinkSnapshot{ATSRunning:true,ATSForeground:true,ManualArmed:true,IPCReady:true,
  PhysicalPacketSent:true,PhysicalCallbackAck:true,HeldDirection:"GAUCHE"}
 d.observe(time.Now(),snap,c,nil,1)
 if d.report.LastInGame==nil || d.report.LastInGame.Result!="CALLBACK_ACK_ONLY_WHEELS_NOT_PROVEN" {
  t.Fatalf("ACK cannot mean physics validated: %+v",d.report.LastInGame)
 }
}

func TestOriginal20PreservesBothDirectionSamplesAndSignedPackets(t *testing.T) {
 destination:=filepath.Join(t.TempDir(),"direction-bidirectional.json")
 recorder:=newPhysicalDiagnosticRecorderAt(destination)
 now:=time.Date(2026,11,21,20,30,0,0,time.UTC)
 conditions:=allPhysicalConditions()
 left:=inputLinkSnapshot{
  ATSRunning:true, ATSForeground:true,ManualArmed:true,IPCReady:true,
  PhysicalPacketSent:true,PhysicalCallbackAck:true,
  HeldDirection:"GAUCHE",PlannedAngleDegrees:-13,
  PhysicalPacketNormalized:-.216667,PhysicalDetectedTrailerID:"ats.test.trailer",
 }
 recorder.observe(now,left,conditions,nil,1)
 right:=left
 right.HeldDirection="DROITE"
 right.PlannedAngleDegrees=14
 right.PhysicalPacketNormalized=.233333
 recorder.observe(now.Add(time.Second),right,conditions,nil,1)
 if recorder.report.LastWithLeftInput==nil || recorder.report.LastWithRightInput==nil {
  t.Fatal("both directions must be retained independently")
 }
 if recorder.report.LastWithLeftInput.NormalizedIPCAngle>=0 ||
    recorder.report.LastWithRightInput.NormalizedIPCAngle<=0 {
  t.Fatalf("signed IPC incorrectly recorded: left=%+v right=%+v",recorder.report.LastWithLeftInput,recorder.report.LastWithRightInput)
 }
 if recorder.report.LastWithLeftInput.ATSDetectedTrailerID !=
    recorder.report.LastWithRightInput.ATSDetectedTrailerID {
  t.Fatal("comparison must refer to the same trailer")
 }
 focusLost:=right
 focusLost.ATSForeground=false
 focusLost.HeldDirection=""
 recorder.observe(now.Add(2*time.Second),focusLost,conditions,nil,1)
 focusLost.ATSRunning=false
 recorder.observe(now.Add(3*time.Second),focusLost,conditions,nil,0)
 raw,err:=os.ReadFile(destination)
 if err!=nil {t.Fatal(err)}
 var loaded physicalDiagnosticReport
 if err:=json.Unmarshal(raw,&loaded);err!=nil {t.Fatal(err)}
 if loaded.LastWithLeftInput==nil || loaded.LastWithRightInput==nil {
  t.Fatal("focus or exit erased the side-by-side evidence")
 }
 // Starting a new ATS game session clears stale comparisons.
 recorder.observe(now.Add(4*time.Second),left,conditions,nil,1)
 if recorder.report.LastWithRightInput!=nil {
  t.Fatal("right evidence from a previous ATS session was retained")
 }
}
