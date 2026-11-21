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
func TestOriginal16GateReportsEveryBlocker(t *testing.T) {
 reasons:=physicalGateReasons(experimentalPhysicalConditions{},false,false)
 if len(reasons)!=9 {t.Fatalf("want 9 blockers; got %d %v",len(reasons),reasons)}
 if len(physicalGateReasons(allPhysicalConditions(),true,true))!=0 {t.Fatal("all valid should have no blockers")}
 cond:=allPhysicalConditions()
 cond.FreshSCSTelemetry=false
 v:=physicalGateReasons(cond,true,true)
 if len(v)!=1 || v[0]!="TELEMETRIE_ABSENTE_EN_PAUSE_OU_PERIMEE" {t.Fatalf("unexpected %v",v)}
}

func TestOriginal16PersistsLastFocusedGameResultAfterAltTabAndExit(t *testing.T) {
 dest:=filepath.Join(t.TempDir(),"ATS-Physical-Test","diagnostic-original16.json")
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
func TestOriginal16NeverReportsAckAsPhysicalSteering(t *testing.T) {
 d:=newPhysicalDiagnosticRecorderAt(filepath.Join(t.TempDir(),"report.json"))
 c:=allPhysicalConditions()
 snap:=inputLinkSnapshot{ATSRunning:true,ATSForeground:true,ManualArmed:true,IPCReady:true,
  PhysicalPacketSent:true,PhysicalCallbackAck:true,HeldDirection:"GAUCHE"}
 d.observe(time.Now(),snap,c,nil,1)
 if d.report.LastInGame==nil || d.report.LastInGame.Result!="CALLBACK_ACK_ONLY_WHEELS_NOT_PROVEN" {
  t.Fatalf("ACK cannot mean physics validated: %+v",d.report.LastInGame)
 }
}
