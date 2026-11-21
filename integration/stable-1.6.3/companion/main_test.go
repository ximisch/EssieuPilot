package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

func TestCompanionNeverWritesStableProfiles(t *testing.T) {
    path:=filepath.Join(t.TempDir(),"profiles.json")
    orig:=[]byte(`{"schema":1,"active_id":"custom","profiles":[{"id":"custom","toggle_key":"KEY:F4","trailers":[{"id":"t","axles":[{"enabled":true,"max_angle":30,"turn_rate":12,"center_rate":7,"ratio":1}]}]}]}`)
    if err:=os.WriteFile(path,orig,0600);err!=nil {t.Fatal(err)}
    p:=profileStore{path:path}
    for i:=0;i<3;i++ { if _,err:=p.read();err!=nil {t.Fatal(err)} }
    after,err:=os.ReadFile(path)
    if err!=nil {t.Fatal(err)}
    if string(after)!=string(orig) {t.Fatal("Stable user profile was changed")}
}
func TestMissingProfileUsesInMemoryDefaults(t *testing.T){
    path:=filepath.Join(t.TempDir(),"none.json")
    p:=profileStore{path:path}
    raw,err:=p.read()
    if err!=nil {t.Fatal(err)}
    var prof runtimeProfiles
    if err:=json.Unmarshal(raw,&prof);err!=nil {t.Fatal(err)}
    if len(prof.Profiles)!=1 || prof.Profiles[0].ToggleKey!="KEY:F8" {t.Fatalf("Invalid default: %+v",prof)}
    if _,err:=os.Stat(path);!os.IsNotExist(err){t.Fatal("Fallback wrote to original profile directory")}
}
func TestInvalidProfileFallback(t *testing.T){
    path:=filepath.Join(t.TempDir(),"profiles.json")
    if err:=os.WriteFile(path,[]byte("invalid json"),0600);err!=nil {t.Fatal(err)}
    p:=profileStore{path:path}
    raw,err:=p.read()
    if err!=nil || !json.Valid(raw) {t.Fatal("No in-memory fallback available")}
    unchanged,_:=os.ReadFile(path)
    if string(unchanged)!="invalid json" {t.Fatal("Invalid original profile was overwritten")}
}
