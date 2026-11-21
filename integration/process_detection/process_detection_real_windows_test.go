//go:build windows

package main

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// Actual Windows Toolhelp process test, without injecting any ATS DLL or
// depending on the user's game installation. A renamed Windows cmd.exe
// simulates only the process name; it does not simulate game telemetry.
func TestGameProcessRunningRealWindows(t *testing.T) {
    if gameProcessRunning() {
        t.Skip("amtrucks.exe already running on this Windows machine")
    }
    src := filepath.Join(os.Getenv("WINDIR"), "System32", "cmd.exe")
    original, err := os.ReadFile(src)
    if err != nil { t.Fatalf("Cannot read Windows cmd.exe: %v", err) }
    dest := filepath.Join(t.TempDir(), "amtrucks.exe")
    if err := os.WriteFile(dest, original, 0700); err != nil {
        t.Fatalf("Cannot prepare named Windows test process: %v", err)
    }
    // ping takes roughly 25 seconds and cmd.exe remains alive during the test.
    cmd := exec.Command(dest, "/D", "/C", "ping 127.0.0.1 -n 26 > NUL")
    if err := cmd.Start(); err != nil {
        t.Fatalf("Cannot start ATS-named test process: %v", err)
    }
    defer func() {
        _ = cmd.Process.Kill()
        _ = cmd.Wait()
    }()
    until := time.Now().Add(8 * time.Second)
    detected := false
    for time.Now().Before(until) {
        if gameProcessRunning() { detected = true; break }
        time.Sleep(80*time.Millisecond)
    }
    if !detected {
        t.Fatalf("Windows process scanner failed to detect renamed amtrucks.exe (PID %d)", cmd.Process.Pid)
    }
    t.Logf("ATS-NAMED PROCESS DETECTED: PID %d", cmd.Process.Pid)
    if err := cmd.Process.Kill(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "finished") {
        t.Fatalf("Cannot stop test process: %v", err)
    }
    _ = cmd.Wait()
    until = time.Now().Add(8 * time.Second)
    for time.Now().Before(until) {
        if !gameProcessRunning() {
            t.Log("ATS-NAMED PROCESS EXIT DETECTED")
            return
        }
        time.Sleep(80*time.Millisecond)
    }
    t.Fatal("Windows process scanner still reports ATS after test process ended")
}
