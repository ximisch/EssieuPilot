# EssieuPilot — detection ATS en lecture seule, sans DLL et sans installation.
# Lance dans PowerShell : .\Watch-ATS.ps1
# Ctrl+C pour arreter.
$ErrorActionPreference = 'Stop'
$previous = $null
Write-Host 'EssieuPilot - surveillance de American Truck Simulator (amtrucks.exe)' -ForegroundColor Cyan
Write-Host 'Ouvre et ferme ATS pour verifier les deux changements d etat. Ctrl+C pour quitter.'
while ($true) {
    $found = @(Get-Process -Name 'amtrucks' -ErrorAction SilentlyContinue)
    $running = ($found.Count -gt 0)
    if ($null -eq $previous -or $running -ne $previous) {
        $moment = Get-Date -Format 'HH:mm:ss'
        if ($running) {
            $pidList = ($found | ForEach-Object { $_.Id }) -join ', '
            Write-Host "[$moment] ATS LANCE - amtrucks.exe detecte (PID: $pidList)" -ForegroundColor Green
        } else {
            Write-Host "[$moment] ATS NON LANCE - aucun processus amtrucks.exe" -ForegroundColor Yellow
        }
        $previous = $running
    }
    Start-Sleep -Seconds 1
}
