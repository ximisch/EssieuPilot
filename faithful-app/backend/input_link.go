package main

import (
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
	ManualArmed            bool   `json:"manual_armed"`
	LeftPressed            bool   `json:"left_pressed"`
	RightPressed           bool   `json:"right_pressed"`
	LastInput              string `json:"last_input"`
	LastEventTime          string `json:"last_event_time"`
	PhysicalControlEnabled bool   `json:"physical_control_enabled"`
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
