"""Only integrates process detection in reconstructed development sources.
Does NOT modify or replace the official 1.6.2 binary or its profiles.
"""
from pathlib import Path
import sys
root=Path(sys.argv[1] if len(sys.argv)>1 else ".").resolve()

def replace(path,before,after):
    f=root/path
    s=f.read_text(encoding="utf-8")
    n=s.count(before)
    if n!=1:
        raise RuntimeError(f"Refused ambiguous patch {path}: {n} occurrences of {before[:55]!r}")
    f.write_text(s.replace(before,after,1),encoding="utf-8")

main=Path("backend/main.go")
replace(main,'router.HandleFunc("GET /api/ats/installations", s.getATSInstallations)',
'router.HandleFunc("GET /api/ats/installations", s.getATSInstallations)\n\trouter.HandleFunc("GET /api/ats/process", s.getATSProcess)')
replace(main,'func (s *server) getATSInstallations(w http.ResponseWriter, r *http.Request) {',
'''// ATS process detection only: no Steam search, telemetry DLL or trailer hooks needed.
var atsProcessProbe = gameProcessRunning

func (s *server) getATSProcess(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "no-store")
    jsonReply(w, http.StatusOK, map[string]any{
        "running": atsProcessProbe(),
        "process": "amtrucks.exe",
    })
}

func (s *server) getATSInstallations(w http.ResponseWriter, r *http.Request) {''')
js=Path("backend/web/modern.js")
existing=(root/js).read_text(encoding="utf-8")
anchor=("  loadATSInstallations();loadPluginState();" if "  loadATSInstallations();loadPluginState();" in existing else
        "  document.getElementById('scanATS').addEventListener('click',loadATSInstallations);loadATSInstallations();")
replace(js,anchor,anchor+'''
  // Process monitor, independent of SCS SDK and of whether ATS is installed via Steam.
  let atsProcessRunning=null, atsTelemetryConnected=false, atsPollBusy=false;
  function renderATSProcess(){
    const running=atsProcessRunning===true;
    const connected=running&&atsTelemetryConnected;
    const label=running?'ATS LANCÉ':(atsProcessRunning===false?'ATS NON LANCÉ':'ÉTAT INDISPONIBLE');
    const mainStatus=document.getElementById('modernGameState');
    if(mainStatus)mainStatus.textContent=connected?'● ATS CONNECTÉ (LECTURE)':'● '+label;
    const diag=document.getElementById('atsInstallation');
    if(diag)diag.textContent=label;
    const sidebar=document.getElementById('sideConnection');
    if(sidebar)sidebar.textContent=connected?'ATS CONNECTÉ (LECTURE)':label;
    const detail=document.getElementById('sideConnectionDetail');
    if(detail)detail.textContent=connected?'Télémétrie active · pas de contrôle physique validé':
        (running?'Jeu détecté sans DLL':'Lecture ATS inactive');
  }
  async function pollATSProcess(){
    if(atsPollBusy)return;
    atsPollBusy=true;
    try{
      const response=await fetch('/api/ats/process',{cache:'no-store'});
      if(!response.ok)throw new Error('process HTTP '+response.status);
      const data=await response.json();
      if(typeof data.running!=='boolean'||data.process!=='amtrucks.exe')throw Error('Format process invalide');
      atsProcessRunning=data.running;
    }catch(_){atsProcessRunning=null;} // Unknown is never falsely called "ATS closed"
    finally{atsPollBusy=false;renderATSProcess();}
  }
  pollATSProcess();
  setInterval(pollATSProcess,1000);''')
replace(js,"    const connected=!!t?.connected;\n    const tr=t?.trailers?.[0];",
            "    const connected=!!t?.connected;\n    atsTelemetryConnected=connected;\n    renderATSProcess();\n    const tr=t?.trailers?.[0];")
def replace_line_start(path,prefix,replacement):
    f=root/path
    lines=f.read_text(encoding="utf-8").splitlines(keepends=True)
    matches=[i for i,l in enumerate(lines) if l.startswith(prefix)]
    if len(matches)!=1:raise RuntimeError(f"Expected one line {prefix!r}, got {len(matches)}")
    lines[matches[0]]=replacement+"\n"
    f.write_text("".join(lines),encoding="utf-8")
replace_line_start(js,"    if(status)status.textContent=connected?","    if(status)renderATSProcess();")
replace_line_start(js,"    const side=document.getElementById('sideConnection');if(side)side.textContent=",
                   "    // Sidebar uses the independent ATS process monitor.")
replace_line_start(js,"    const sideDet=document.getElementById('sideConnectionDetail');if(sideDet)sideDet.textContent=",
                   "    // Status of the telemetry DLL is independent from process status.")
print("ATS launch/stop detection integrated in reconstructed developer UI. NO STABLE RELEASE.")
