package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestControlModeEndpointRejectsMissingToken(t *testing.T) {
    srv:= &server{token:"test-token"}
    req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1/api/control-link/mode",strings.NewReader(`{"enabled":false}`))
    res:=httptest.NewRecorder()
    srv.setInputMode(res,req)
    if res.Code != http.StatusForbidden { t.Fatalf("expected 403, got %d",res.Code) }
}

func TestControlModeEndpointRejectsMissingOrMalformedInput(t *testing.T) {
    srv:= &server{token:"test-token"}
    for _,body := range []string{`{}`,`{"enabled":"yes"}`,`{"test":true}`,`not-json`} {
        req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1/api/control-link/mode",strings.NewReader(body))
        req.Header.Set("X-ATS-Token","test-token")
        res:=httptest.NewRecorder()
        srv.setInputMode(res,req)
        if res.Code != http.StatusBadRequest {t.Fatalf("invalid input %q accepted: %d",body,res.Code)}
    }
}

func TestGUIRequestDisablesTestModeWithoutPhysicalSteering(t *testing.T) {
    srv:= &server{token:"test-token"}
    req:=httptest.NewRequest(http.MethodPost,"http://127.0.0.1/api/control-link/mode",strings.NewReader(`{"enabled":false}`))
    req.Header.Set("X-ATS-Token","test-token")
    res:=httptest.NewRecorder()
    srv.setInputMode(res,req)
    if res.Code != http.StatusOK || !strings.Contains(res.Body.String(),`"physical_control_enabled":false`) {
        t.Fatalf("GUI test-mode endpoint allowed unsafe physical control: %d %s",res.Code,res.Body.String())
    }
}
