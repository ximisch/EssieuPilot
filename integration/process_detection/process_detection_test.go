package main

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "testing"
)

// Proves that ATS process presence is reported even without a telemetry plugin.
func TestATSProcessEndpointNoDLL(t *testing.T) {
 old:=atsProcessProbe
 defer func(){atsProcessProbe=old}()
 for _,tc:=range []struct{name string; running bool}{{"started",true},{"closed",false}} {
  t.Run(tc.name,func(t *testing.T){
   atsProcessProbe=func()bool{return tc.running}
   rec:=httptest.NewRecorder()
   req:=httptest.NewRequest(http.MethodGet,"http://127.0.0.1/api/ats/process",nil)
   (&server{}).getATSProcess(rec,req)
   if rec.Code!=http.StatusOK {t.Fatalf("Unexpected status %d",rec.Code)}
   if rec.Header().Get("Cache-Control")!="no-store" {t.Fatal("Cache must be disabled")}
   var v struct {Running bool `json:"running"`; Process string `json:"process"`}
   if err:=json.Unmarshal(rec.Body.Bytes(),&v);err!=nil {t.Fatal(err)}
   if v.Running!=tc.running || v.Process!="amtrucks.exe" {t.Fatalf("Incorrect process status %+v",v)}
  })
 }
}
