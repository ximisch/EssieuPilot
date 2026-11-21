package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Diagnostic of configurable input -> IPC preparation. An IPC buffer is not
// evidence that a DLL is loaded, nor that ATS steering works.
type inputLinkSnapshot struct {
	ServiceRunning         bool   `json:"service_running"`
	IPCReady               bool   `json:"ipc_ready"`
	ProfileName            string `json:"profile_name"`
	ToggleBinding          string `json:"toggle_binding"`
	LeftBinding            string `json:"left_binding"`
	RightBinding           string `json:"right_binding"`
	AnalogBinding          string `json:"analog_binding"`
	ProfileSaved           bool   `json:"profile_saved"`
	ATSForeground          bool   `json:"ats_foreground"`
	ForegroundPID          uint32 `json:"foreground_pid"`
	ATSProcessPID          uint32 `json:"ats_process_pid"`
	FocusReason            string `json:"focus_reason"`
	InputReason            string `json:"input_reason"`
	InputMonitoringActive  bool   `json:"input_monitoring_active"`
	LeftVirtualKey         uint16 `json:"left_virtual_key"`
	RightVirtualKey        uint16 `json:"right_virtual_key"`
	ManualArmed            bool   `json:"manual_armed"`
	ATSRunning             bool   `json:"ats_running"`
	ActivationCount        uint64 `json:"activation_count"`
	LeftCount              uint64 `json:"left_count"`
	RightCount             uint64 `json:"right_count"`
	AllowedLeftCount       uint64 `json:"allowed_left_count"`
	AllowedRightCount      uint64 `json:"allowed_right_count"`
	SteeringInputAllowed   bool   `json:"steering_input_allowed"`
	HeldDirection          string `json:"held_direction"`
	RequestedPercent       float64 `json:"requested_percent"`
	RequestedSource        string  `json:"requested_source"`
	AnalogDetectedPercent  float64 `json:"analog_detected_percent"`
	AnalogAuthorizedPercent float64 `json:"analog_authorized_percent"`
	LastAuthorizedCommand  string  `json:"last_authorized_command"`
	LastAuthorizedTime     string  `json:"last_authorized_time"`
	AnalogLeftCount        uint64  `json:"analog_left_count"`
	AnalogRightCount       uint64  `json:"analog_right_count"`
	LastDirection          string `json:"last_direction"`
	LastDirectionTime      string `json:"last_direction_time"`
	LastFocusTime          string `json:"last_focus_time"`
	InputHistory           []string `json:"input_history"`
	LeftPressed            bool   `json:"left_pressed"`
	RightPressed           bool   `json:"right_pressed"`
	LastInput              string `json:"last_input"`
	LastEventTime          string `json:"last_event_time"`
	PhysicalControlEnabled bool   `json:"physical_control_enabled"`
	PhysicalDLLLoaded      bool   `json:"physical_dll_loaded"`
	PhysicalDLLSDKAccepted bool   `json:"physical_dll_sdk_accepted"`
	PhysicalDLLGameVersion string `json:"physical_dll_game_version"`
	PhysicalDLLProbeMessage string `json:"physical_dll_probe_message"`
	PhysicalExperimentOptIn bool `json:"physical_experiment_opt_in"`
	PhysicalPacketSent bool `json:"physical_packet_sent"`
	PhysicalPacketNormalized float64 `json:"physical_packet_normalized"`
	PhysicalCallbackAck bool `json:"physical_callback_ack"`
	PhysicalCallbackAckAgeMs uint64 `json:"physical_callback_ack_age_ms"`
	PhysicalValidationStatus string `json:"physical_validation_status"`
	PlannedAngleDegrees    float64 `json:"planned_angle_degrees"`
	PlannedAngleValid      bool    `json:"planned_angle_valid"`
	PlannedAngleReason     string  `json:"planned_angle_reason"`
	ObservedSpeedKMH       float64 `json:"observed_speed_kmh"`
	ObservedSpeedValid     bool    `json:"observed_speed_valid"`
	ObservedWheelSteeringValid bool `json:"observed_wheel_steering_valid"`
	ObservedWheelSteeringDegrees []float64 `json:"observed_wheel_steering_degrees"`
	DLLConfirmed           bool   `json:"dll_confirmed"`
	StatusMessage          string `json:"status_message"`
}

var inputLinkState = struct {
	sync.RWMutex
	value inputLinkSnapshot
}{
	value: inputLinkSnapshot{
		StatusMessage: "Préparation des touches configurables ; aucune commande physique n'est autorisée.",
	},
}

func publishInputLinkSnapshot(v inputLinkSnapshot) {
	inputLinkState.Lock()
	inputLinkState.value = v
	inputLinkState.Unlock()
}
func snapshotInputLink() inputLinkSnapshot {
	inputLinkState.RLock()
	result := inputLinkState.value
	inputLinkState.RUnlock()
	return result
}
func (s *server) getInputLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	jsonReply(w, 200, snapshotInputLink())
}
func formatInputEvent(name string, at time.Time) (string, string) {
	if name == "" {
		return "", ""
	}
	return name, at.Format(time.RFC3339)
}

// User-confirmed local UI command, separate from the game input. This changes
// ONLY the test-mode diagnostic; the IPC to ATS is always disabled in stage 1.
var inputModeRequests = struct {
	sync.Mutex
	sequence uint64
	enabled bool
}{}

func readInputModeRequest(lastSequence uint64) (uint64, bool, bool) {
	inputModeRequests.Lock()
	defer inputModeRequests.Unlock()
	if inputModeRequests.sequence == lastSequence {
		return lastSequence, false, false
	}
	return inputModeRequests.sequence, inputModeRequests.enabled, true
}

func (s *server) setInputMode(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		jsonReply(w, http.StatusForbidden, map[string]string{"error":"Commande refusée"})
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 512))
	if err != nil {
		jsonReply(w, http.StatusBadRequest, map[string]string{"error":"Commande invalide"})
		return
	}
	var cmd struct { Enabled *bool `json:"enabled"` }
	if err := json.Unmarshal(body, &cmd); err != nil || cmd.Enabled == nil {
		jsonReply(w, http.StatusBadRequest, map[string]string{"error":"Mode de test invalide"})
		return
	}
	if *cmd.Enabled && !statusATSRunning() {
		jsonReply(w, http.StatusConflict, map[string]string{"error":"Lance ATS avant d'activer le contrôleur de test."})
		return
	}
	inputModeRequests.Lock()
	inputModeRequests.sequence++
	inputModeRequests.enabled = *cmd.Enabled
	inputModeRequests.Unlock()
	jsonReply(w, http.StatusOK, map[string]any{
		"ok": true, "enabled": *cmd.Enabled,
		"physical_control_enabled": false,
		"message": "Mode de diagnostic modifié, direction des roues non activée.",
	})
}
