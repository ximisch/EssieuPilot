"""Apply the ATS runtime diagnostics to reconstructed sources after UI patches."""
from pathlib import Path

root=Path(__file__).resolve().parent

def patch(path, updates, tail=''):
    f=root/path
    s=f.read_text(encoding='utf-8')
    for a,b in updates:
        if s.count(a)!=1:
            raise RuntimeError(f"Patch mismatch in {path}: {a!r} appears {s.count(a)} times")
        s=s.replace(a,b,1)
    f.write_text(s+tail,encoding='utf-8')

patch('backend/ats_windows.go', [
    ('"strings"\n)', '"strings"\n    "syscall"\n    "unsafe"\n)'),
    ('foundPlugin := false\n\t\tentries, _ := os.ReadDir(plugins)', 'foundPlugin := false\n        conflictingDLLs := []string{}\n\t\tentries, _ := os.ReadDir(plugins)'),
    ('if !entry.IsDir() && strings.HasSuffix(name, ".dll") && (strings.Contains(name, "telemetry") || strings.Contains(name, "scs")) {\n\t\t\t\tfoundPlugin = true\n\t\t\t\tbreak\n\t\t\t}',
     'if entry.IsDir() || !strings.HasSuffix(name, ".dll") { continue }\n            if name == "scs_telemetry_hub.dll" { foundPlugin = true }\n            if name != "scs_telemetry_hub.dll" && (strings.Contains(name, "telemetry") || strings.Contains(name, "scs_sdk") || strings.Contains(name, "westplay")) { conflictingDLLs = append(conflictingDLLs, entry.Name()) }'),
    ('Une DLL de télémétrie possible est présente (fonctionnement à tester).','SCS Telemetry Hub présent (connexion à vérifier).'),
    ('Pas de DLL de télémétrie reconnue ; aucun accès direct à la physique.','SCS Telemetry Hub absent : aucune télémétrie. Aucun accès direct à la physique.'),
    ('"plugin_found": foundPlugin, "message": status, "path": game}', '"plugin_found": foundPlugin, "running": gameProcessRunning(), "other_plugin_dlls": conflictingDLLs, "message": status, "path": game}'),
    ('"plugin_found": false, "message": fmt.Sprintf', '"plugin_found": false, "running": gameProcessRunning(), "message": fmt.Sprintf'),
], '''
// Detect the ATS process independently of the telemetry plugin.
// Uses read-only standard Windows process enumeration API.
const th32csSnapprocess = 0x00000002
type processEntry32W struct {
    Size uint32
    Usage uint32
    ProcessID uint32
    DefaultHeapID uintptr
    ModuleID uint32
    Threads uint32
    ParentProcessID uint32
    BasePriority int32
    Flags uint32
    ExeFile [260]uint16
}
func gameProcessRunning() bool {
    kernel := syscall.NewLazyDLL("kernel32.dll")
    snap := kernel.NewProc("CreateToolhelp32Snapshot")
    first := kernel.NewProc("Process32FirstW")
    next := kernel.NewProc("Process32NextW")
    closeHandle := kernel.NewProc("CloseHandle")
    handle, _, _ := snap.Call(th32csSnapprocess, 0)
    if handle == ^uintptr(0) { return false }
    defer closeHandle.Call(handle)
    p := processEntry32W{Size: uint32(unsafe.Sizeof(processEntry32W{}))}
    ok, _, _ := first.Call(handle, uintptr(unsafe.Pointer(&p)))
    for ok != 0 {
        if strings.EqualFold(syscall.UTF16ToString(p.ExeFile[:]), "amtrucks.exe") { return true }
        ok, _, _ = next.Call(handle, uintptr(unsafe.Pointer(&p)))
    }
    return false
}
''')
patch('backend/ats_other.go',[], '\nfunc gameProcessRunning() bool { return false }\n')
patch('backend/telemetry.go', [
    ('Connected       bool              `json:"connected"`','Connected       bool              `json:"connected"`\n    GameRunning     bool              `json:"game_running"`'),
    ('out := telemetryResult{Trailers: []detectedTrailer{}, Source: "SCS Telemetry Hub (optionnel)", Note: "Aucune télémétrie ATS accessible. Le module SCS Telemetry Hub doit être installé et ATS démarré. Aucun pilotage physique actif."}',
     '''out := telemetryResult{Trailers: []detectedTrailer{}, GameRunning: gameProcessRunning(), Source: "SCS Telemetry Hub (optionnel)", Note: "Aucune télémétrie ATS accessible. Le module SCS Telemetry Hub doit être installé et ATS démarré. Aucun pilotage physique actif."}
    if out.GameRunning {
        out.Note = "ATS est lancé, mais la télémétrie reste inactive. Vérifie scs_telemetry_hub.dll et les anciens plugins."
    }'''),
])
patch('backend/web/modern.js', [
    ("if(status)status.textContent=connected?'● Connecté (lecture)':'● Non connecté';", "if(status)status.textContent=connected?'● ATS connecté (lecture)':(t?.game_running?'● ATS lancé (télémétrie inactive)':'● ATS non lancé / inaccessible');"),
    ("badge.textContent=connected?'TÉLÉMÉTRIE ACTIVE':'NON CONNECTÉ'", "badge.textContent=connected?'TÉLÉMÉTRIE ACTIVE':(t?.game_running?'ATS DÉTECTÉ · DLL À VÉRIFIER':'NON CONNECTÉ')"),
    ("side.textContent=connected?'ATS CONNECTÉ (LECTURE)':'ATS NON CONNECTÉ'", "side.textContent=connected?'ATS CONNECTÉ (LECTURE)':(t?.game_running?'ATS LANCÉ · DLL INACTIVE':'ATS NON CONNECTÉ')"),
    ("sideDet.textContent=connected?'Physique non pilotée':'Télémétrie SCS compatible requise'", "sideDet.textContent=connected?'Physique non pilotée':(t?.game_running?'DLL de télémétrie non détectée':'Télémétrie SCS compatible requise')"),
    ("document.getElementById('atsInstallation').textContent=j.found?'Trouvé':'Non localisé';", "document.getElementById('atsInstallation').textContent=j.running?'ATS LANCÉ':(j.found?'Installé (fermé)':'Non localisé');"),
    ("document.getElementById('atsTelemetryPlugin').textContent=j.plugin_found?'DLL présente':'DLL non détectée';", "document.getElementById('atsTelemetryPlugin').textContent=j.plugin_found?'Hub détecté':'Hub absent';"),
])
patch('backend/telemetry_test.go',[],'''
func TestUnavailableTelemetryNeverClaimsPhysicalControl(t *testing.T) {
    b:=newTelemetryBridge()
    defer b.close()
    b.endpoint="http://127.0.0.1:1/snapshot"
    got:=b.read()
    if got.Connected || got.PhysicalControl || len(got.Trailers)>0 {
        t.Fatalf("Disconnected telemetry must not claim steering: %+v", got)
    }
}
''')
print('ATS telemetry and process status diagnostics applied.')
