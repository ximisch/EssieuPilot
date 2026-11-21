package main

// Original.14: persist a concise R&D diagnostic while ATS is actually focused.
// This is a read-only observation of the command gate, not proof of wheel motion.
import (
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "strings"
 "time"
)

const physicalDiagnosticFilename = "diagnostic-original14.json"
const physicalDiagnosticMaxEvents = 40

type physicalDiagnosticFrame struct {
 At string `json:"at"`
 Result string `json:"result"`
 Blockers []string `json:"blockers"`
 ModeManual bool `json:"mode_manual"`
 ExplicitConsent bool `json:"consent_loaded_at_app_start"`
 DLLReady bool `json:"dll_sdk_recognized"`
 IPCReady bool `json:"ipc_ready"`
 TelemetryFresh bool `json:"telemetry_fresh_and_unpaused"`
 TrailerCount int `json:"trailer_count"`
 ProfileMatched bool `json:"profile_matched"`
 AxlesCompatible bool `json:"axles_compatible"`
 RequestedDirection string `json:"requested_direction,omitempty"`
 AngleTargetDegrees float64 `json:"target_angle_degrees"`
 ATSDetectedTrailerID string `json:"ats_detected_trailer_id,omitempty"`
 SelectedPresetID string `json:"selected_preset_id,omitempty"`
 SelectionReason string `json:"selection_reason,omitempty"`
 NormalizedIPCAngle float64 `json:"normalized_ipc_angle"`
 PhysicalFullScaleDegrees float64 `json:"calibration_degrees_at_full_lock,omitempty"`
 IndependentAxlePhysics bool `json:"independent_axle_physics_supported"`
 IPCSent bool `json:"ipc_packet_sent"`
 CallbackACK bool `json:"dll_callback_ack"`
 ObservedWheelSteeringDegrees []float64 `json:"ats_sdk_measured_wheel_angles_degrees,omitempty"`
}

type physicalDiagnosticReport struct {
 Schema int `json:"schema"`
 Build string `json:"build"`
 Warning string `json:"warning"`
 SessionStarted string `json:"session_started,omitempty"`
 LastGameSeen string `json:"last_game_seen,omitempty"`
 LastGameStopped string `json:"last_game_stopped,omitempty"`
 LastInGame *physicalDiagnosticFrame `json:"last_in_game,omitempty"`
 LastWithManualInput *physicalDiagnosticFrame `json:"last_with_manual_input,omitempty"`
 SentCount int `json:"ipc_sent_samples"`
 ACKCount int `json:"callback_ack_samples"`
 Events []physicalDiagnosticFrame `json:"events"`
}

type physicalDiagnosticRecorder struct {
 path string
 report physicalDiagnosticReport
 lastKey string
 lastFlush time.Time
 gameWasRunning bool
}

func physicalDiagnosticPath() (string,error) {
 dir,err:=os.UserCacheDir()
 if err!=nil || dir=="" {return "",errors.New("LOCALAPPDATA introuvable")}
 return filepath.Join(dir,"EssieuPilot","ATS-Physical-Test",physicalDiagnosticFilename),nil
}

func newPhysicalDiagnosticRecorder() *physicalDiagnosticRecorder {
 path,err:=physicalDiagnosticPath()
 if err!=nil {return nil}
 return newPhysicalDiagnosticRecorderAt(path)
}
func newPhysicalDiagnosticRecorderAt(path string) *physicalDiagnosticRecorder {
 return &physicalDiagnosticRecorder{
  path:path,
  report:physicalDiagnosticReport{Schema:1,Build:"Original.14",
   Warning:"Rapport IPC/SDK de recherche. Reference de calibrage non mesuree, valeur ATS normalisee non equivalente aux degres reels. Un ACK ne prouve pas le mouvement des roues. Pilotage independant des essieux NON disponible. SOLO uniquement.",
   Events:make([]physicalDiagnosticFrame,0)},
 }
}

// Return all blocking conditions, instead of the single generic 'authorization
// missing' that Original.11 displayed after Alt+Tab.
func physicalGateReasons(c experimentalPhysicalConditions, bridgeReady, planReady bool) []string {
 reasons:=make([]string,0,9)
 if !c.ExplicitResearchOptIn {reasons=append(reasons,"AUTORISATION_EXPERIMENTALE_ABSENTE_AU_DEMARRAGE")}
 if !c.InputAuthorized {reasons=append(reasons,"MODE_MANUEL_OU_FOCUS_ATS_INACTIF")}
 if !c.DLLSDKProbeRecognized {reasons=append(reasons,"DLL_SDK_NON_RECONNU")}
 if !c.FreshSCSTelemetry {reasons=append(reasons,"TELEMETRIE_ABSENTE_EN_PAUSE_OU_PERIMEE")}
 if !c.OneAttachedTrailer {reasons=append(reasons,"REMORQUE_UNIQUE_NON_DETECTEE")}
 if !c.ActiveProfileMatched {reasons=append(reasons,"PROFIL_REMORQUE_NON_ASSOCIE")}
 if !c.CompatibleAxles {reasons=append(reasons,"GROUPE_ESSIEUX_INCOMPATIBLE")}
 if !bridgeReady {reasons=append(reasons,"CANAL_IPC_NON_PRET")}
 if !planReady {reasons=append(reasons,"ANGLE_CIBLE_NON_DISPONIBLE")}
 return reasons
}

func (d *physicalDiagnosticRecorder) observe(now time.Time, snapshot inputLinkSnapshot, c experimentalPhysicalConditions, reasons []string, trailerCount int) {
 if d==nil {return}
 if !snapshot.ATSRunning {
  if d.gameWasRunning {
   d.gameWasRunning=false
   d.report.LastGameStopped=now.UTC().Format(time.RFC3339)
   d.lastKey=""
   d.flush(now)
  }
  return
 }
 if !d.gameWasRunning {
  d.gameWasRunning=true
  d.report.SessionStarted=now.UTC().Format(time.RFC3339)
  d.report.LastGameStopped=""
  // New ATS session: don't retain older results as if they were this run.
  d.report.Events=nil
  d.report.LastInGame=nil
  d.report.LastWithManualInput=nil
  d.report.SentCount=0
  d.report.ACKCount=0
  d.lastKey=""
 }
 // Deliberately never override the last in-game result on Alt+Tab:
 // the game is no longer focused and telemetry often pauses.
 if !snapshot.ATSForeground {return}
 result:="BLOCKED"
 if snapshot.PhysicalPacketSent {
  result="IPC_SENT_NO_ACK"
  if snapshot.PhysicalCallbackAck {result="CALLBACK_ACK_ONLY_WHEELS_NOT_PROVEN"}
 } else if len(reasons)==0 {
  result="READY_BUT_PACKET_NOT_SENT"
 }
 requested:=snapshot.HeldDirection
 if requested=="" && snapshot.ManualArmed {requested="AUCUNE_TOUCHE_MAINTENUE"}
 frame:=physicalDiagnosticFrame{
  At:now.UTC().Format(time.RFC3339Nano),Result:result,
  Blockers:append([]string{},reasons...),
  ModeManual:snapshot.ManualArmed,
  ExplicitConsent:c.ExplicitResearchOptIn,
  DLLReady:c.DLLSDKProbeRecognized,
  IPCReady:snapshot.IPCReady,
  TelemetryFresh:c.FreshSCSTelemetry,
  TrailerCount:trailerCount,
  ProfileMatched:c.ActiveProfileMatched,
  AxlesCompatible:c.CompatibleAxles,
  RequestedDirection:requested,
  AngleTargetDegrees:snapshot.PlannedAngleDegrees,
  ATSDetectedTrailerID:snapshot.PhysicalDetectedTrailerID,
  SelectedPresetID:snapshot.PhysicalSelectedTrailerID,
  SelectionReason:snapshot.PhysicalSelectionReason,
  NormalizedIPCAngle:snapshot.PhysicalPacketNormalized,
  PhysicalFullScaleDegrees:snapshot.PhysicalCalibrationReferenceDegrees,
  IndependentAxlePhysics:false,
  IPCSent:snapshot.PhysicalPacketSent,
  CallbackACK:snapshot.PhysicalCallbackAck,
 }
 if snapshot.ObservedWheelSteeringValid {
  frame.ObservedWheelSteeringDegrees=append([]float64(nil),snapshot.ObservedWheelSteeringDegrees...)
 }
 d.report.LastGameSeen=now.UTC().Format(time.RFC3339)
 d.report.LastInGame=&frame
 if snapshot.HeldDirection=="GAUCHE" || snapshot.HeldDirection=="DROITE" {
  d.report.LastWithManualInput=&frame
 }
 // Prevent huge logs: capture state changes or one sample every 3s.
 key:=fmt.Sprintf("%s|%t|%s|%t|%t|%s",result,snapshot.ManualArmed,snapshot.HeldDirection,
  snapshot.PhysicalPacketSent,snapshot.PhysicalCallbackAck,strings.Join(reasons,","))
 if key==d.lastKey && now.Sub(d.lastFlush)<3*time.Second{return}
 d.lastKey=key
 if snapshot.PhysicalPacketSent {d.report.SentCount++}
 if snapshot.PhysicalCallbackAck {d.report.ACKCount++}
 d.report.Events=append(d.report.Events,frame)
 if len(d.report.Events)>physicalDiagnosticMaxEvents {
  d.report.Events=append([]physicalDiagnosticFrame(nil),
   d.report.Events[len(d.report.Events)-physicalDiagnosticMaxEvents:]...)
 }
 d.flush(now)
}

func (d *physicalDiagnosticRecorder) flush(now time.Time) {
 if d==nil {return}
 if err:=os.MkdirAll(filepath.Dir(d.path),0700);err!=nil{return}
 data,err:=json.MarshalIndent(d.report,"","  ")
 if err!=nil{return}
 data=append(data,'\n')
 temp,err:=os.CreateTemp(filepath.Dir(d.path),"report-*.tmp")
 if err!=nil{return}
 name:=temp.Name()
 defer os.Remove(name)
 if _,err=temp.Write(data);err!=nil {temp.Close();return}
 if err=temp.Close();err!=nil{return}
 if err=os.Rename(name,d.path);err!=nil {
  // Windows AV may temporarily hold the previous JSON open.
  if err=os.WriteFile(d.path,data,0600);err!=nil{return}
 }
 d.lastFlush=now
}

// Download the exact recorded file from the app's Diagnostic screen. This
// endpoint has no ability to activate the physical plugin.
func (s *server) getPhysicalDiagnostic(w http.ResponseWriter,r *http.Request) {
 path,err:=physicalDiagnosticPath()
 if err!=nil {http.Error(w,"Dossier de diagnostic indisponible",500);return}
 data,err:=os.ReadFile(path)
 if err!=nil {http.Error(w,"Aucun essai ATS enregistré depuis Original.14",404);return}
 if len(data)>200000 || !json.Valid(data) {http.Error(w,"Rapport de diagnostic invalide",500);return}
 w.Header().Set("Content-Type","application/json; charset=utf-8")
 w.Header().Set("Content-Disposition",`attachment; filename="EssieuPilot_Diagnostic_Physique_Original12.json"`)
 w.Header().Set("Cache-Control","no-store")
 _,_=w.Write(data)
}
