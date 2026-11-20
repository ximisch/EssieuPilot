package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoveredProfilesAreCopiedNotModified(t *testing.T) {
	tmp := t.TempDir()
	original := filepath.Join(tmp, "EssieuPilot", "profiles.json")
	path := filepath.Join(tmp, "EssieuPilot-Reconstruction", "profiles.json")
	if err := os.MkdirAll(filepath.Dir(original), 0700); err != nil {
		t.Fatal(err)
	}
	source := []byte(`{"schema":1,"active_id":"x","unknown":"keep me","profiles":[{"id":"x","trailers":[{"axles":[{"id":"a","custom":123}]}]}]}`)
	if err := os.WriteFile(original, source, 0600); err != nil {
		t.Fatal(err)
	}
	store := profileStore{path: path, previous: original}
	copied, err := store.read()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(copied, source) {
		t.Fatalf("profile changed: %s", copied)
	}
	if err = store.write([]byte(strings.ReplaceAll(string(source), "123", "321"))); err != nil {
		t.Fatal(err)
	}
	intact, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(intact, source) {
		t.Fatal("Original installed user profile was changed!")
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, source) {
		t.Fatal("The reconstruction backup does not equal pre-save profiles")
	}
}

func TestProfileValidation(t *testing.T) {
	for _, input := range []string{`{}`, `{"schema":2,"profiles":[]}`, `{"schema":1,"profiles":[]}`, `{"schema":1,"profiles":[{"trailers":[]}]}`, `{"schema":1,"profiles":[{"trailers":[{"axles":[]}]}]}`} {
		if err := validateProfiles([]byte(input)); err == nil {
			t.Errorf("accepted invalid profiles: %s", input)
		}
	}
	if err := validateProfiles(sampleProfiles()); err != nil {
		t.Fatal(err)
	}
}

func TestAppAndProtectedWrites(t *testing.T) {
	p := profileStore{path: filepath.Join(t.TempDir(), "profiles.json")}
	s, err := newServer(p)
	if err != nil {
		t.Fatal(err)
	}
	check := func(method, path, token string, body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://127.0.0.1:12345"+path, bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:55000"
		if token != "" {
			req.Header.Set("X-ATS-Token", token)
		}
		rec := httptest.NewRecorder()
		s.http.Handler.ServeHTTP(rec, req)
		return rec
	}
	if rec := check("GET", "/", "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), s.token) {
		t.Fatal("Original HTML or anti-CSRF token missing")
	}
	if rec := check("GET", "/app.js", "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "SteeringLogic") {
		t.Fatal("Original front-end not served")
	}
	if rec := check("GET", "/api/config", "", nil); rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	if rec := check("POST", "/api/config", "invalid", sampleProfiles()); rec.Code != 403 {
		t.Fatal("Write accepted without authorization")
	}
	if rec := check("POST", "/api/config", s.token, sampleProfiles()); rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	if rec := check("GET", "/api/updates", "", nil); rec.Code != 200 {
		t.Fatal(rec.Body.String())
	} else {
		var v map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v["available"] != false {
			t.Fatal("Unverified update installer was enabled")
		}
	}
	if rec := check("POST", "/api/updates/install", s.token, nil); rec.Code != 409 {
		t.Fatal("Unverified installer was allowed")
	}
}
