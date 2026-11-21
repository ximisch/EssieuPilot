package main
import "net/http"

// Read-only status endpoint, never authorizes multiplayer physics and never
// communicates with a server or peer.
func (s *server) getConvoyProbe(w http.ResponseWriter,r *http.Request) {
    report:=readConvoyProbe()
    jsonReply(w,http.StatusOK,map[string]any{
        "game_running": statusATSRunning(),
        "report":report,
        "other_players_see_steering": "NON_VERIFIE",
        "physical_convoy_supported": false,
        "truckersmp_supported": false,
    })
}
