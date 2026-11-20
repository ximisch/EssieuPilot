// EssieuPilot reconstructed backend. This is an isolated development build,
// NOT a replacement for the installed v1.6.2 and NOT a physical steering mod.
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var frontend embed.FS

const technicalVersion = "2.0.0-original.4"

type server struct {
	telemetry telemetryBridge
	token     string
	store     *profileStore
	http      *http.Server
	mu        sync.Mutex
}

type profileStore struct {
	path     string
	previous string
	mu       sync.Mutex
}

func profilePaths() (profileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return profileStore{}, err
	}
	return profileStore{
		path:     filepath.Join(dir, "EssieuPilot-OriginalTest", "profiles.json"),
		previous: filepath.Join(dir, "EssieuPilot", "profiles.json"),
	}, nil
}

func sampleProfiles() json.RawMessage {
	return json.RawMessage(`{"schema":1,"active_id":"profile-1","profiles":[{"id":"profile-1","name":"Profil par défaut","toggle_key":"KEY:F8","left_key":"KEY:F9","right_key":"KEY:F10","analog_axis":"","analog_deadzone":0.08,"analog_curve":1,"analog_mode":"position","analog_full_scale":100,"analog_output_pct":100,"match_key":"","auto_center_moving":true,"center_speed_gain":1,"trailers":[{"id":"trailer-1","name":"Semi-remorque 1","model_type":"surbaissée","axles":[{"id":"axle-1","name":"Essieu directeur 1","enabled":true,"max_angle":35,"turn_rate":14,"center_rate":7,"ratio":1,"reversed":false,"current_angle":0},{"id":"axle-2","name":"Essieu directeur 2","enabled":true,"max_angle":35,"turn_rate":14,"center_rate":7,"ratio":1,"reversed":false,"current_angle":0},{"id":"axle-3","name":"Essieu directeur 3","enabled":true,"max_angle":35,"turn_rate":14,"center_rate":7,"ratio":1,"reversed":false,"current_angle":0}]}]}]}`)
}

func validateProfiles(content []byte) error {
	if len(content) < 10 || len(content) > 4<<20 {
		return errors.New("taille de profil invalide")
	}
	var obj struct {
		Schema   int               `json:"schema"`
		Profiles []json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal(content, &obj); err != nil {
		return err
	}
	if obj.Schema != 1 || len(obj.Profiles) < 1 || len(obj.Profiles) > 100 {
		return errors.New("format de profils ou nombre de profils invalide")
	}
	for _, p := range obj.Profiles {
		var pr struct {
			Trailers []json.RawMessage `json:"trailers"`
		}
		if err := json.Unmarshal(p, &pr); err != nil {
			return err
		}
		if len(pr.Trailers) < 1 || len(pr.Trailers) > 20 {
			return errors.New("nombre de remorques invalide")
		}
		for _, t := range pr.Trailers {
			var tr struct {
				Axles []json.RawMessage `json:"axles"`
			}
			if err := json.Unmarshal(t, &tr); err != nil {
				return err
			}
			if len(tr.Axles) < 1 || len(tr.Axles) > 32 {
				return errors.New("nombre d'essieux invalide")
			}
		}
	}
	return nil
}

func (s *profileStore) read() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if content, err := os.ReadFile(s.path); err == nil {
		if e := validateProfiles(content); e != nil {
			return nil, fmt.Errorf("profils de reconstruction invalides: %w", e)
		}
		return content, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// Copy *once*, without changing the existing v1.6.2 installation's file.
	if content, err := os.ReadFile(s.previous); err == nil && validateProfiles(content) == nil {
		if err = writeAtomic(s.path, content); err != nil {
			return nil, err
		}
		return content, nil
	}
	content := sampleProfiles()
	if err := writeAtomic(s.path, content); err != nil {
		return nil, err
	}
	return content, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".profiles-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *profileStore) write(data []byte) error {
	if err := validateProfiles(data); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if previous, err := os.ReadFile(s.path); err == nil {
		// Keep a restorable snapshot before overwriting the reconstruction's own file.
		if err := writeAtomic(s.path+".bak", previous); err != nil {
			return err
		}
	}
	return writeAtomic(s.path, data)
}

func newServer(p profileStore) (*server, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, err
	}
	s := &server{token: hex.EncodeToString(key[:]), store: &p, telemetry: newTelemetryBridge()}
	router := http.NewServeMux()
	router.HandleFunc("GET /", s.index)
	router.HandleFunc("GET /styles.css", s.static)
	router.HandleFunc("GET /modern.css", s.static)
	router.HandleFunc("GET /app.js", s.static)
	router.HandleFunc("GET /modern.js", s.static)
    router.HandleFunc("GET /original-live.js", s.static)
	router.HandleFunc("GET /steering_logic.js", s.static)
	router.HandleFunc("GET /logo.png", s.static)
	router.HandleFunc("GET /hero_reference.jpg", s.static)
	for i := 0; i < 5; i++ {
		router.HandleFunc(fmt.Sprintf("GET /feature_%d.jpg", i), s.static)
	}
	router.HandleFunc("GET /api/config", s.getProfiles)
	router.HandleFunc("POST /api/config", s.saveProfiles)
	router.HandleFunc("GET /api/devices", s.getDevices)
	router.HandleFunc("GET /api/telemetry", s.getTelemetry)
    router.HandleFunc("GET /api/status", s.getLiveStatus)
    router.HandleFunc("GET /api/control-link", s.getInputLink)
    router.HandleFunc("POST /api/control-link/mode", s.setInputMode)
	router.HandleFunc("GET /api/ats/installations", s.getATSInstallations)
	router.HandleFunc("GET /api/updates", s.getOriginalUpdates)
	router.HandleFunc("POST /api/updates/install", s.installOriginalUpdate)
	router.HandleFunc("POST /api/quit", s.quit)
	s.http = &http.Server{Handler: s.secure(router), ReadHeaderTimeout: 5 * time.Second}
	return s, nil
}

func (s *server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "127.0.0.1" && !strings.HasPrefix(r.Host, "127.0.0.1:") {
			http.Error(w, "Loopback only", http.StatusForbidden)
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(ip) == nil || !net.ParseIP(ip).IsLoopback() {
			http.Error(w, "Local access only", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1:") {
			http.Error(w, "Origin not allowed", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *server) authorized(r *http.Request) bool {
	provided := r.Header.Get("X-ATS-Token")
	if len(provided) != len(s.token) {
		return false
	}
	var diff byte
	for i := 0; i < len(s.token); i++ {
		diff |= s.token[i] ^ provided[i]
	}
	return diff == 0
}
func jsonReply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func jsonError(w http.ResponseWriter, status int, message string) {
	jsonReply(w, status, map[string]string{"error": message})
}
func (s *server) index(w http.ResponseWriter, r *http.Request) {
	// Root only; no duplicate unknown path falling through the top-level handler.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	buf, err := fs.ReadFile(frontend, "web/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	html := strings.ReplaceAll(string(buf), "__TOKEN__", s.token)
	// The original front end remains intact; only the existing technical version
	// footer is replaced and a status-poll script is appended.
	html = strings.ReplaceAll(html, "Version technique 1.6.2", "Version technique "+technicalVersion)
	html = strings.Replace(html, "</body>", `<script src="/original-live.js" defer></script></body>`, 1)
	buf = []byte(html)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf)
}
func (s *server) static(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if strings.ContainsAny(name, "/\\") {
		http.NotFound(w, r)
		return
	}
	buf, err := fs.ReadFile(frontend, "web/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch filepath.Ext(name) {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg":
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Write(buf)
}
func (s *server) getProfiles(w http.ResponseWriter, r *http.Request) {
	content, err := s.store.read()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(content)
}
func (s *server) saveProfiles(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		jsonError(w, 403, "Commande refusée")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		jsonError(w, 413, "Profils trop volumineux")
		return
	}
	if err = s.store.write(body); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	jsonReply(w, 200, map[string]any{"ok": true})
}
func (s *server) getDevices(w http.ResponseWriter, r *http.Request) { jsonReply(w, 200, readDevices()) }
func (s *server) getTelemetry(w http.ResponseWriter, r *http.Request) {
	// Reading a local SDK bridge is OBSERVATION ONLY; it does not steer axles.
	jsonReply(w, 200, s.telemetry.read())
}
func (s *server) getATSInstallations(w http.ResponseWriter, r *http.Request) {
	jsonReply(w, 200, scanATS())
}
func (s *server) getUpdates(w http.ResponseWriter, r *http.Request) {
	jsonReply(w, 200, map[string]any{"version": technicalVersion, "feed_configured": false, "available": false, "download_ready": false, "message": "Reconstruction en développement : installation de mises à jour désactivée jusqu'à validation Windows/ATS."})
}
func (s *server) installUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		jsonError(w, 403, "Commande refusée")
		return
	}
	jsonError(w, 409, "Les mises à jour de la reconstruction ne sont pas activées : aucun installateur publié.")
}
func (s *server) quit(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		jsonError(w, 403, "Commande refusée")
		return
	}
	jsonReply(w, 200, map[string]any{"ok": true})
	go func() { time.Sleep(250 * time.Millisecond); _ = s.http.Close() }()
}
func main() {
	p, err := profilePaths()
	if err != nil {
		log.Fatal(err)
	}
	s, err := newServer(p)
	if err != nil {
		log.Fatal(err)
	}
	// Stage 1: detect dynamically configured key/gamepad bindings and maintain
	// the neutral IPC channel. No physical steering activation on ATS 1.62.
	stopControl := startSteeringRuntime(s)
	defer stopControl()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("READY http://127.0.0.1:%d/\n", listener.Addr().(*net.TCPAddr).Port)
	if runtime.GOOS != "windows" {
		log.Printf("Development server: http://127.0.0.1:%d/", listener.Addr().(*net.TCPAddr).Port)
	}
	if err := s.http.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
