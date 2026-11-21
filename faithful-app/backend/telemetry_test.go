package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEstimatedAxlesAreNotWheels(t *testing.T) {
	x := []position{{Z: 3.00}, {Z: 3.01}, {Z: 4.50}, {Z: 4.51}, {Z: 6.0}, {Z: 6.03}}
	if n := estimateAxleCount(x); n != 3 {
		t.Fatalf("6 wheels should be 3 axles, got %d", n)
	}
	if n := estimateAxleCount(nil); n != 0 {
		t.Fatalf("missing wheels should be unknown, got %d", n)
	}
}

func TestLocalTelemetryHubReadsATSButNeverClaimsSteering(t *testing.T) {
	const snapshot = `{"type":"snapshot","game":"ats","data":{"speed_mps":10,"trailer":{"connected":true}},"trailer_spec":{"id":"lowboy_special","name":"Lowboy Heavy","body_type":"lowboy","wheel_positions":[{"z":2},{"z":2.01},{"z":3.5},{"z":3.51}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, snapshot) }))
	defer srv.Close()
	bridge := newTelemetryBridge()
	defer bridge.close()
	bridge.endpoint = strings.Replace(srv.URL, "http://", "http://", 1)
	x := bridge.read()
	if !x.Connected || x.SpeedKMH != 36 || len(x.Trailers) != 1 {
		t.Fatalf("bridge failed to identify ATS: %+v", x)
	}
	if x.Trailers[0].ID != "lowboy_special" || x.Trailers[0].EstimatedAxles != 2 || x.Trailers[0].SteerableAxlesKnown || x.PhysicalControl {
		t.Fatalf("invalid trailer detection/steering claim: %+v", x)
	}
}

func TestDetachedTrailerIsNotReassociatedFromStaleSpec(t *testing.T) {
	const snap = `{"type":"snapshot","game":"ats","data":{"speed_mps":0,"trailer":{"connected":false}},"trailer_spec":{"id":"previous_trailer","wheel_positions":[{"z":1}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, snap) }))
	defer srv.Close()
	b := newTelemetryBridge()
	defer b.close()
	b.endpoint = srv.URL
	r := b.read()
	if !r.Connected || len(r.Trailers) != 0 {
		t.Fatalf("detached trailer mistakenly detected: %+v", r)
	}
}

func TestRejectETSTelemetryAndUntrustedEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"type":"snapshot","game":"ets2","data":{}}`)
	}))
	defer srv.Close()
	b := newTelemetryBridge()
	defer b.close()
	b.endpoint = srv.URL
	if r := b.read(); r.Connected {
		t.Fatalf("ETS2 wrongly detected as ATS: %+v", r)
	}
	b.endpoint = "http://example.org/snapshot"
	if r := b.read(); r.Connected || !strings.Contains(r.Note, "non autorisée") {
		t.Fatalf("nonloopback endpoint accepted: %+v", r)
	}
}

func TestUnknownTrailerAxlesMustNotBeAssumed(t *testing.T) {
	const snapshot = `{"type":"snapshot","game":"ats","data":{"speed_mps":0,"trailer":{"connected":true}},"trailer_spec":{"id":"mod.trailer","name":"Remorque mod","wheel_positions":[]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, snapshot) }))
	defer srv.Close()
	b := newTelemetryBridge()
	defer b.close()
	b.endpoint = srv.URL
	got := b.read()
	if !got.Connected || len(got.Trailers) != 1 || got.Trailers[0].EstimatedAxles != 0 || got.Trailers[0].SteerableAxlesKnown || got.PhysicalControl {
		t.Fatalf("cannot infer or control steering on unknown mod: %+v", got)
	}
}

func TestTelemetryHubRejectsMalformedAndUnsafeReadings(t *testing.T) {
	for _, snapshot := range []string{
		`not json`,
		`{"type":"event","game":"ats","data":{}}`,
		`{"type":"snapshot","game":"ats","data":{"speed_mps":180}}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, snapshot) }))
		b := newTelemetryBridge()
		b.endpoint = srv.URL
		got := b.read()
		b.close()
		srv.Close()
		if got.Connected || got.PhysicalControl || len(got.Trailers) != 0 {
			t.Fatalf("untrustworthy snapshot treated as ATS connected: %+v", got)
		}
	}
}

// Regression: the actual SCS Telemetry Hub JSON field is `game_paused`,
// not `paused`. Verified against plugin/src/lib.rs Snapshot in upstream.
func TestTelemetryPausedMatchesRealHubSchema(t *testing.T) {
	const snapshot = `{"type":"snapshot","game":"ats","data":{"game_paused":true,"speed_mps":0,"trailer":{"connected":true}},"trailer_spec":{"id":"trailer.unknown","name":"Known ID, unknown axles"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, snapshot) }))
	defer srv.Close()
	bridge := newTelemetryBridge()
	defer bridge.close()
	bridge.endpoint = srv.URL
	got := bridge.read()
	if !got.Connected || !got.Paused || got.SpeedKMH != 0 || len(got.Trailers) != 1 || got.Trailers[0].EstimatedAxles != 0 || got.PhysicalControl {
		t.Fatalf("actual hub game_paused field misread or unknown wheel data invented: %+v", got)
	}
}

func TestTrailerWheelSteeringTelemetryIsIndependentOfPhysicalCommand(t *testing.T) {
	const snapshot = `{"type":"snapshot","game":"ats","data":{"speed_mps":0,"game_paused":false,"trailer":{"connected":true,"wheels":[{"steering":0.0291666666667},{"steering":0.0291666666667},{"steering":0}]}},"trailer_spec":{"id":"scs.lowboy","wheel_positions":[{"z":2},{"z":2.01},{"z":3}]}}`
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,snapshot)}))
	defer srv.Close()
	b:=newTelemetryBridge();defer b.close();b.endpoint=srv.URL
	got:=b.read()
	if !got.Connected||!got.WheelSteeringValid||len(got.WheelSteeringDegrees)!=3||
		got.WheelSteeringDegrees[0]!=10.5||got.WheelSteeringDegrees[2]!=0||
		got.PhysicalControl {
		t.Fatalf("wheel telemetry incorrectly parsed or treated as physical command: %+v",got)
	}
}

func TestTrailerWheelTelemetryNotInventedOnDetachedOrInvalidReadings(t *testing.T) {
	for _,tc:=range []string{
		`{"type":"snapshot","game":"ats","data":{"trailer":{"connected":false,"wheels":[{"steering":0.03}]}},"trailer_spec":{"id":"old","wheel_positions":[{"z":2}]}}`,
		`{"type":"snapshot","game":"ats","data":{"trailer":{"connected":true,"wheels":[{"steering":0.5}]}},"trailer_spec":{"id":"current","wheel_positions":[{"z":2}]}}`,
		`{"type":"snapshot","game":"ats","data":{"trailer":{"connected":true,"wheels":[{"steering":0.03}]}},"trailer_spec":{"id":"current","wheel_positions":[]}}`,
	}{
		srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,tc)}))
		b:=newTelemetryBridge();b.endpoint=srv.URL
		got:=b.read()
		b.close();srv.Close()
		if got.WheelSteeringValid||len(got.WheelSteeringDegrees)>0 {
			t.Fatalf("untrustworthy wheel motion invented from telemetry: %+v",got)
		}
	}
}
