package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Read-only bridge to an optional SCS telemetry SDK plugin. The plugin's
// snapshot API is described at:
// https://github.com/UncappingBadger/Badgers-SCS-Telemetry-Hub
// The SCS telemetry SDK reports data but cannot command physical steering.
const telemetrySnapshot = "http://127.0.0.1:38471/snapshot"

type telemetryBridge struct {
	client   *http.Client
	endpoint string
}

type telemetryResult struct {
	Connected       bool              `json:"connected"`
	Paused          bool              `json:"paused"`
	SpeedKMH        float64           `json:"speed_kmh"`
	Trailers        []detectedTrailer `json:"trailers"`
	Source          string            `json:"source"`
	Note            string            `json:"note"`
	PhysicalControl bool              `json:"physical_control"`
	WheelSteeringDegrees []float64   `json:"wheel_steering_degrees"`
	WheelSteeringValid   bool        `json:"wheel_steering_valid"`
}

type detectedTrailer struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	EstimatedAxles      int    `json:"estimated_axles"`
	AxleCountSource     string `json:"axle_count_source"`
	SteerableAxlesKnown bool   `json:"steerable_axles_known"`
}

type position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type hubSnapshot struct {
	Type string `json:"type"`
	Game string `json:"game"`
	Data struct {
		SpeedMPS float64 `json:"speed_mps"`
		Paused   bool    `json:"game_paused"`
		Trailer struct {
			Connected bool `json:"connected"`
			Wheels []struct {
				Steering float64 `json:"steering"`
			} `json:"wheels"`
		} `json:"trailer"`
	} `json:"data"`
	TrailerSpec struct {
		ID             string     `json:"id"`
		Name           string     `json:"name"`
		BodyType       string     `json:"body_type"`
		WheelPositions []position `json:"wheel_positions"`
	} `json:"trailer_spec"`
}

func newTelemetryBridge() telemetryBridge {
	transport := &http.Transport{
		Proxy:             nil, // loopback must not use proxy settings
		DialContext:       (&net.Dialer{Timeout: 180 * time.Millisecond}).DialContext,
		DisableKeepAlives: true,
	}
	return telemetryBridge{
		client:   &http.Client{Transport: transport, Timeout: 220 * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		endpoint: telemetrySnapshot,
	}
}

// wheel_positions is a collection of wheels, not axles. Two wheels sharing
// a longitudinal position represent ONE axle. Do not infer steerability.
func estimateAxleCount(wheels []position) int {
	if len(wheels) == 0 || len(wheels) > 256 {
		return 0
	}
	z := make([]float64, 0, len(wheels))
	for _, p := range wheels {
		if math.IsNaN(p.Z) || math.IsInf(p.Z, 0) || math.Abs(p.Z) > 500 {
			return 0
		}
		z = append(z, p.Z)
	}
	sort.Float64s(z)
	count := 0
	last := math.Inf(-1)
	for _, pos := range z {
		// The simulator reports left/right wheels at nearly the same Z.
		if pos-last > 0.25 {
			count++
			last = pos
		}
	}
	if count > 32 {
		return 0
	}
	return count
}

func (b telemetryBridge) read() telemetryResult {
	out := telemetryResult{Trailers: []detectedTrailer{}, Source: "SCS Telemetry Hub (optionnel)", Note: "Aucune télémétrie ATS accessible. Le module SCS Telemetry Hub doit être installé et ATS démarré. Aucun pilotage physique actif."}
	endpoint, err := url.Parse(b.endpoint)
	if err != nil || endpoint.Scheme != "http" || endpoint.Hostname() != "127.0.0.1" || endpoint.User != nil || endpoint.Fragment != "" {
		out.Note = "Source de télémétrie non autorisée."
		return out
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, b.endpoint, nil)
	if err != nil {
		out.Note = "Requête de télémétrie invalide."
		return out
	}
	res, err := b.client.Do(req)
	if err != nil {
		return out
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		out.Note = fmt.Sprintf("Source de télémétrie indisponible (HTTP %d).", res.StatusCode)
		return out
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20+1))
	if err != nil || len(body) > 1<<20 {
		out.Note = "Données de télémétrie illisibles ou trop volumineuses."
		return out
	}
	var s hubSnapshot
	if err = json.Unmarshal(body, &s); err != nil {
		out.Note = "Format de télémétrie invalide."
		return out
	}
	if s.Type != "snapshot" || !strings.EqualFold(s.Game, "ats") {
		out.Note = "Une source de télémétrie est présente, mais elle ne fournit pas un état ATS valide."
		return out
	}
	if math.IsNaN(s.Data.SpeedMPS) || math.IsInf(s.Data.SpeedMPS, 0) || math.Abs(s.Data.SpeedMPS) > 120 {
		out.Note = "Vitesse ATS incohérente."
		return out
	}
	out.Connected = true
	out.Paused = s.Data.Paused
	out.SpeedKMH = s.Data.SpeedMPS * 3.6
	out.Note = "ATS connecté en lecture seule par télémétrie SCS. Les essieux ne sont PAS commandés physiquement."
	if !s.Data.Trailer.Connected {
		return out
	}
	id := strings.TrimSpace(s.TrailerSpec.ID)
	if id == "" || len(id) > 160 {
		out.Note += " Remorque attelée, mais son identifiant est indisponible ; aucune association automatique."
		return out
	}
	name := strings.TrimSpace(s.TrailerSpec.Name)
	if name == "" {
		name = id
	}
	if len(name) > 160 {
		name = name[:160]
	}
	bodyType := strings.TrimSpace(s.TrailerSpec.BodyType)
	if len(bodyType) > 80 {
		bodyType = bodyType[:80]
	}
	if bodyType == "" {
		bodyType = "inconnu"
	}
	axles := estimateAxleCount(s.TrailerSpec.WheelPositions)
	source := "non fourni par le SDK"
	if axles > 0 {
		source = "estimation via position des roues"
	}
	out.Trailers = append(out.Trailers, detectedTrailer{ID: id, Name: name, Type: bodyType, EstimatedAxles: axles, AxleCountSource: source, SteerableAxlesKnown: false})
	// The hub reports trailer.wheel.steering in rotations, not degrees.
	// Compare these independent, read-only SDK measurements with the
	// requested command in R&D; zero readings or an SDK value cannot
	// prove that the wheel model visibly animated.
	wheels := s.Data.Trailer.Wheels
	if len(wheels) > 0 && len(wheels) <= 20 && len(s.TrailerSpec.WheelPositions) > 0 {
		angles := make([]float64,0,len(wheels))
		valid := true
		for _,wheel := range wheels {
			if math.IsNaN(wheel.Steering) || math.IsInf(wheel.Steering,0) ||
				math.Abs(wheel.Steering) > 0.25 {
				valid = false
				break
			}
			angles = append(angles,math.Round(wheel.Steering*36000)/100)
		}
		if valid {
			out.WheelSteeringDegrees = angles
			out.WheelSteeringValid = true
		}
	}
	return out
}

func (b telemetryBridge) close() error {
	if b.client == nil {
		return errors.New("client absent")
	}
	b.client.CloseIdleConnections()
	return nil
}
