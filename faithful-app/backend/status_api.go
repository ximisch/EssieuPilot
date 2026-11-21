package main

import "net/http"

// The two lamps report independent facts. Process detection does not require
// an ATS SDK DLL. The DLL ACK means an invocation occurred, not verified
// visual/physical wheel movement or multiplayer compatibility.
func (s *server) getLiveStatus(w http.ResponseWriter, r *http.Request) {
    game := statusATSRunning()
    jsonReply(w, http.StatusOK, map[string]any{
        "game_running": game,
        "steering_ack": game && statusSteeringACK(),
        "physical_wheel_movement_verified": false,
    })
}
