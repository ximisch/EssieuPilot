"""Attach the Windows-only Go IPC controller to the recovered backend.
Fail if the original source changes; never modify the stable 1.6.2.
"""
from pathlib import Path
import re
main=Path("backend/main.go")
data=main.read_text(encoding="utf-8")
needle=r'(\ts, err := newServer\(p\)\n\tif err != nil \{\n\t\tlog.Fatal\(err\)\n\t\}\n)'
replacement=r'\1\tstopSteering := startSteeringRuntime(s)\n\tdefer stopSteering()\n'
changed,n=re.subn(needle,replacement,data)
if n!=1:
    raise SystemExit(f"Expected one newServer() initialization, got {n}; refusing unsafe patch")
main.write_text(changed,encoding="utf-8")
print("Backend bridge lifecycle enabled (stops and clears IPC on shutdown).")
