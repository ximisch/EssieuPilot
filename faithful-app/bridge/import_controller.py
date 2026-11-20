"""Port the verified reference IPC and dynamic binding reader into the original UI backend.
Stage 1 DOES NOT enable physical steering. Every IPC packet has enabled=false.
"""
from pathlib import Path

root=Path(__file__).resolve().parents[1]
reference=root/"steering-reference"
backend=root/"backend"
names=[
    "steering_protocol.go",
    "steering_protocol_test.go",
    "steering_bridge_windows.go",
    "steering_bridge_other.go",
    "steering_runtime_windows.go",
    "steering_runtime_other.go",
]
for name in names:
    target=backend/name
    if target.exists():
        raise RuntimeError(f"Refusing to overwrite controller file: {target}")
    target.write_bytes((reference/name).read_bytes())

runtime=backend/"steering_runtime_windows.go"
text=runtime.read_text("utf-8")
before='var getAsyncKey = epKernel32.NewProc("GetAsyncKeyState")\nvar getForeground = epKernel32.NewProc("GetForegroundWindow")\nvar getWindowPID = epKernel32.NewProc("GetWindowThreadProcessId")'
after='var epUser32 = syscall.NewLazyDLL("user32.dll")\nvar getAsyncKey = epUser32.NewProc("GetAsyncKeyState")\nvar getForeground = epUser32.NewProc("GetForegroundWindow")\nvar getWindowPID = epUser32.NewProc("GetWindowThreadProcessId")'
if text.count(before)!=1:
    raise RuntimeError("Cannot locate prototype input library bug")
text=text.replace(before,after)
marker="func startSteeringRuntime(s *server) func() {"
if text.count(marker)!=1:
    raise RuntimeError("Cannot locate prototype runtime")
text=text.split(marker)[0]+r'''
// Stage-1 input routing: read the saved active profile every 250ms.
// Uses the same mapping format as the real physical runtime: KEY:* and
// DEVICE:*; the physical DLL never hardcodes F8/F9/F10.
func startSteeringRuntime(s *server) func() {
    bridge, bridgeErr := newPhysicalSteeringBridge()
    if bridgeErr != nil {
        log.Printf("EssieuPilot IPC preparation unavailable: %v", bridgeErr)
    }
    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan struct{})
    go func() {
        defer close(done)
        if bridge != nil {defer bridge.close()}
        ticker := time.NewTicker(40*time.Millisecond)
        defer ticker.Stop()
        var profiles runtimeProfiles
        var lastProfileRead time.Time
        var lastInput string
        var lastEventTime time.Time
        var manual, prevToggle, prevLeft, prevRight, prevAnalog bool
        var profileID string
        var profileLoaded bool
        for {
            select {
            case <-ctx.Done():
                if bridge != nil {_ = bridge.publish(false,0,0)}
                publishInputLinkSnapshot(inputLinkSnapshot{})
                return
            case <-ticker.C:
            }
            now := time.Now()
            if now.Sub(lastProfileRead) >= 250*time.Millisecond {
                lastProfileRead=now
                buf,err:=s.store.read()
                var current runtimeProfiles
                profileLoaded=err==nil && json.Unmarshal(buf,&current)==nil && len(current.Profiles)>0
                if profileLoaded {profiles=current} else {profiles=runtimeProfiles{}}
            }
            active,found:=activeSavedProfile(profiles)
            if !found {profileID=""}
            if active.ID != profileID {
                profileID=active.ID
                manual=false;prevToggle=false;prevLeft=false;prevRight=false;prevAnalog=false
                lastInput="";lastEventTime=time.Time{}
            }
            foreground:=isForegroundATS()
            snapshot:=inputLinkSnapshot{
                ServiceRunning:true, IPCReady:bridge!=nil,
                ProfileSaved:profileLoaded && found,
                ProfileName: active.ID,
                ToggleBinding:active.ToggleKey,
                LeftBinding:active.LeftKey,
                RightBinding:active.RightKey,
                AnalogBinding:active.AnalogAxis,
                ATSForeground:foreground,
                PhysicalControlEnabled:false,
                DLLConfirmed:false,
                StatusMessage:"Touches transmises au contrôleur. Canal IPC neutre : aucun braquage physique activé.",
            }
            if !foreground {
                manual=false;prevToggle=false;prevLeft=false;prevRight=false;prevAnalog=false
                // Only observe user input when ATS owns the foreground window.
            } else if found {
                devices:=readDevices()
                toggle:=deviceBindingDown(active.ToggleKey,devices)
                if toggle && !prevToggle {
                    manual=!manual
                    lastInput="Activation ("+active.ToggleKey+")"
                    if !manual {lastInput="Désactivation ("+active.ToggleKey+")"}
                    lastEventTime=now
                }
                prevToggle=toggle
                left:=deviceBindingDown(active.LeftKey,devices)
                right:=deviceBindingDown(active.RightKey,devices)
                if left && !prevLeft && manual {
                    lastInput="Gauche ("+active.LeftKey+")"
                    lastEventTime=now
                }
                if right && !prevRight && manual {
                    lastInput="Droite ("+active.RightKey+")"
                    lastEventTime=now
                }
                prevLeft=left
                prevRight=right
                analog:=math.Abs(analogFromDevices(active,devices))>=0.1
                if analog && !prevAnalog && manual {
                    lastInput="Axe analogique ("+active.AnalogAxis+")"
                    lastEventTime=now
                }
                prevAnalog=analog
                snapshot.LeftPressed=left && manual
                snapshot.RightPressed=right && manual
            }
            snapshot.ManualArmed=manual
            snapshot.LastInput,snapshot.LastEventTime=formatInputEvent(lastInput,lastEventTime)
            // Important safety interlock: NEVER send an enabled packet to a
            // version-unverified DLL. This first update tests mapping + IPC only.
            if bridge!=nil {
                if err:=bridge.publish(false,0,0);err!=nil {
                    snapshot.IPCReady=false
                    snapshot.StatusMessage="Canal IPC indisponible : "+err.Error()
                }
            }
            publishInputLinkSnapshot(snapshot)
        }
    }()
    return func() {cancel();<-done}
}

func activeSavedProfile(profiles runtimeProfiles) (runtimeProfile,bool) {
    for _,candidate:=range profiles.Profiles {
        if candidate.ID==profiles.ActiveID {return candidate,true}
    }
    return runtimeProfile{},false
}
'''
if "bridge.publish(true" in text:
    raise RuntimeError("Unsafe physical steering enabled in stage 1")
runtime.write_text(text,encoding="utf-8")
print("Dynamic input mapping: Windows USER32 keyboard + gamepad/joystick bindings, stage 1 IPC always neutral")
