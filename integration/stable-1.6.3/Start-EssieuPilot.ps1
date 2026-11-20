# EssieuPilot 1.6.3 — stable original GUI + ATS steering companion
# No WebView2 reconstruction. User's existing stable profiles remain untouched.
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$gui = Join-Path $root "EssieuPilot.exe"
$ctrl = Join-Path $root "EssieuPilotController.exe"
$dir = Join-Path $env:LOCALAPPDATA "EssieuPilot\diagnostics"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$log = Join-Path $dir "launcher-1.6.3.log"
function WriteLog($message) { Add-Content -LiteralPath $log -Value ("[{0}] {1}" -f (Get-Date -Format o),$message) }
$sidecar = $null
$watcher = $null
try {
  if (-not (Test-Path -LiteralPath $gui)) {throw "Application stable EssieuPilot.exe introuvable."}
  if (-not (Test-Path -LiteralPath $ctrl)) {throw "Contrôleur ATS EssieuPilotController.exe introuvable."}

  $sidecar = Start-Process -FilePath $ctrl -WorkingDirectory $root -WindowStyle Hidden -PassThru
  WriteLog "Contrôleur lancé PID=$($sidecar.Id)."
  $main = Start-Process -FilePath $gui -WorkingDirectory $root -PassThru
  WriteLog "Interface EssieuPilot stable lancée PID=$($main.Id)."
  # Show status over the EssieuPilot window without modifying its historical executable.
  $watch = Join-Path $root 'Watch-ATS.ps1'
  if (-not (Test-Path -LiteralPath $watch)) { throw 'Diagnostic ATS introuvable (Watch-ATS.ps1).' }
  $ps = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
  $watchArgs = '-NoProfile -STA -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + $watch + '" -GuiPid ' + $main.Id + ' -ParentPid ' + $PID
  $watcher = Start-Process -FilePath $ps -ArgumentList $watchArgs -WorkingDirectory $root -WindowStyle Hidden -PassThru
  WriteLog "Indicateurs ATS / pilotage lancés PID=$($watcher.Id), interface PID=$($main.Id)."
  # RC4: detect silent PowerShell initialization crashes instead of leaving old labels as-is.
  Start-Sleep -Milliseconds 1200
  $watcher.Refresh()
  if ($watcher.HasExited) {
    $diagnostic=Join-Path $dir 'indicateurs-1.6.3-rc4.log'
    throw "Le panneau des voyants ne s'est pas ouvert. Diagnostic : $diagnostic"
  }
  WriteLog "RC4 indicateurs actifs, consulter le journal du panneau si les voyants ne sont pas visibles."
  $main.WaitForExit()
  WriteLog "Interface fermée, exit=$($main.ExitCode)."
}
catch {
  WriteLog ("Erreur: "+$_.Exception.ToString())
  [System.Windows.Forms.MessageBox]::Show($_.Exception.Message,"EssieuPilot")
}
finally {
  if($null -ne $watcher) {
    try {
      if (-not $watcher.HasExited) {
        # The watcher normally stops on its own when its parent goes away.
        Stop-Process -Id $watcher.Id -ErrorAction Stop
      }
      WriteLog "Diagnostic ATS arrêté."
    } catch { WriteLog ("Arrêt diagnostic: " + $_.Exception.Message) }
  }
  if($null -ne $sidecar) {
    try { if (-not $sidecar.HasExited) { Stop-Process -Id $sidecar.Id -ErrorAction Stop }; WriteLog "Contrôleur arrêté." }
    catch { WriteLog ("Arrêt contrôleur: " + $_.Exception.Message) }
  }
}
