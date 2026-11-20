// EssieuPilot 1.6.3: companion to the original stable Windows executable.
// NO WebView2 UI / localhost server. The existing stable GUI stays intact.
// Uses read-only original profiles and sends commands to an optional ATS DLL.
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "runtime"
    "syscall"
    "time"
    "unsafe"
)

type profileStore struct { path string }
type server struct { store *profileStore; telemetry telemetryBridge }

func stableProfilePath() string {
    dir, err := os.UserConfigDir()
    if err != nil { return "" }
    return filepath.Join(dir, "EssieuPilot", "profiles.json")
}

// Read existing stable profile only. Never overwrite, migrate or replace it.
func (p *profileStore) read() ([]byte,error) {
    if p.path != "" {
        if b,e:=os.ReadFile(p.path);e==nil && len(b)>10 && len(b)<4<<20 {
            var v runtimeProfiles
            if json.Unmarshal(b,&v)==nil && len(v.Profiles)>0 { return b,nil }
        }
    }
    return []byte(defaultProfiles),nil
}
const defaultProfiles = `{"schema":1,"active_id":"default","profiles":[{"id":"default","toggle_key":"KEY:F8","left_key":"KEY:F9","right_key":"KEY:F10","auto_center_moving":true,"trailers":[{"id":"trailer-1","axles":[{"enabled":true,"max_angle":35,"turn_rate":14,"center_rate":7,"ratio":1}]}]}]}`

var controllerMutex = syscall.NewLazyDLL("kernel32.dll").NewProc("CreateMutexW")
func singleInstance() func() {
    name,_:=syscall.UTF16PtrFromString(`LocalEssieuPilotControllerV163`)
    h,_,e:=controllerMutex.Call(0,0,uintptr(unsafe.Pointer(name)))
    if h==0 { log.Printf("Mutex unavailable: %v",e); return func(){} }
    if e==syscall.Errno(183) { epCloseHandle.Call(h); log.Fatal("EssieuPilot controller already running") }
    return func(){ epCloseHandle.Call(h) }
}

func main() {
    if runtime.GOOS!="windows" { fmt.Fprintln(os.Stderr,"Windows only"); return }
    dir,err:=os.UserCacheDir()
    if err==nil {
        dir=filepath.Join(dir,"EssieuPilot","diagnostics")
        if os.MkdirAll(dir,0700)==nil {
            if f,e:=os.OpenFile(filepath.Join(dir,"controller-1.6.3.log"),os.O_CREATE|os.O_WRONLY|os.O_APPEND,0600);e==nil {
                defer f.Close()
                log.SetOutput(f)
            }
        }
    }
    releaseMutex:=singleInstance()
    defer releaseMutex()
    log.Println("EssieuPilot 1.6.3 companion started; original GUI preserved.")
    log.Printf("Read-only profiles: %s",stableProfilePath())
    srv:=&server{store:&profileStore{path:stableProfilePath()},telemetry:newTelemetryBridge()}
    stop:=startSteeringRuntime(srv)
    defer stop()
    defer srv.telemetry.close()
    signals:=make(chan os.Signal,1)
    signal.Notify(signals,os.Interrupt,syscall.SIGTERM)
    ticker:=time.NewTicker(10*time.Second)
    defer ticker.Stop()
    log.Println("Steering bridge initialized; enabled only when ATS foreground and telemetry valid.")
    for {
        select {
        case <-signals: log.Println("Controller stopping"); return
        case <-ticker.C:
            t:=srv.telemetry.read()
            log.Printf("ATS running=%t, telemetry=%t, trailers=%d",t.GameRunning,t.Connected,len(t.Trailers))
        }
    }
}
