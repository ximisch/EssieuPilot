param([ValidateSet("Install","Remove","Status")][string]$Action="Status",[string]$GamePath="")
$ErrorActionPreference="Stop"
Set-StrictMode -Version Latest
function FindGame([string]$Given) {
  if($Given) {$candidates=@($Given)}
  else {
    $candidates=@(
      "D:\Program Files (x86)\Steam\steamapps\common\American Truck Simulator",
      "C:\Program Files (x86)\Steam\steamapps\common\American Truck Simulator",
      "D:\SteamLibrary\steamapps\common\American Truck Simulator",
      "C:\SteamLibrary\steamapps\common\American Truck Simulator",
      "D:\Steam\steamapps\common\American Truck Simulator",
      "C:\Steam\steamapps\common\American Truck Simulator",
      "E:\SteamLibrary\steamapps\common\American Truck Simulator"
    )
    try {
      $steam=(Get-ItemProperty "HKCU:\Software\Valve\Steam").SteamPath.Replace("/","\")
      $candidates=@((Join-Path $steam "steamapps\common\American Truck Simulator"))+$candidates
      $vdf=Join-Path $steam "steamapps\libraryfolders.vdf"
      if(Test-Path -LiteralPath $vdf) {
        $matches=[regex]::Matches((Get-Content -LiteralPath $vdf -Raw),'"path"\s+"([^"]+)"')
        foreach($m in $matches) {$candidates=@((Join-Path $m.Groups[1].Value.Replace("\\","\") "steamapps\common\American Truck Simulator"))+$candidates}
      }
    } catch {}
  }
  foreach($c in $candidates) {
    if(Test-Path -LiteralPath (Join-Path $c "bin\win_x64\amtrucks.exe")) {return (Resolve-Path -LiteralPath $c).Path}
  }
  throw "American Truck Simulator introuvable. Ouvre le script avec -GamePath suivi de ton dossier ATS."
}
function CheckClosed {
  if(Get-Process -Name "amtrucks" -ErrorAction SilentlyContinue){throw "Ferme ATS avant toute modification des DLL."}
}
function GetHash([string]$path) {
  return (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
}
$manifest=Join-Path $PSScriptRoot "essieupilot-installed.json"
$entries=@("EssieuPilotATS.dll","scs_telemetry_hub.dll")
try {
  $previous=$null
  if(Test-Path -LiteralPath $manifest){
    $previous=Get-Content -LiteralPath $manifest -Raw | ConvertFrom-Json
    if(-not $GamePath){$GamePath=[string]$previous.GamePath}
  }
  $game=FindGame $GamePath
  $plugins=Join-Path $game "bin\win_x64\plugins"
  if(Test-Path -LiteralPath $plugins) {
    $dir=Get-Item -LiteralPath $plugins -Force
    if(-not $dir.PSIsContainer -or ($dir.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
      throw "Dossier plugins non standard (lien symbolique ou fichier) : aucune modification."
    }
  }
  if($Action -eq "Status"){
    Write-Host "ATS : $game"
    foreach($name in $entries) {Write-Host "$name : $(Test-Path -LiteralPath (Join-Path $plugins $name))"}
    exit 0
  }
  CheckClosed
  if($Action -eq "Remove") {
    if($null -eq $previous){Write-Host "Aucune DLL enregistrée par EssieuPilot. Les autres plugins sont préservés.";exit 0}
    if($previous.Version -ne 1 -or $previous.GamePath -ne $game -or $previous.Files.Count -ne 2){
      throw "Enregistrement du plugin invalide. Suppression refusée."
    }
    foreach($entry in $previous.Files){
      if($entry.Name -notin $entries){throw "Nom DLL dans le manifeste non reconnu."}
      $target=Join-Path $plugins $entry.Name
      if(Test-Path -LiteralPath $target){
        $file=Get-Item -LiteralPath $target -Force
        if($file.PSIsContainer -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -or (GetHash $target) -ne $entry.SHA256){
          throw "DLL modifiée par un tiers, retrait refusé : $target"
        }
      }
    }
    foreach($entry in $previous.Files) {
      $target=Join-Path $plugins $entry.Name
      if(Test-Path -LiteralPath $target){Remove-Item -LiteralPath $target -ErrorAction Stop}
    }
    Remove-Item -LiteralPath $manifest -ErrorAction Stop
    Write-Host "Deux DLL retirées. Les autres plugins ATS ont été conservés."
    exit 0
  }
  if($null -ne $previous){throw "Déjà installé : utilise le raccourci Retirer les DLL ATS avant de réinstaller."}
  if(Test-Path -LiteralPath (Join-Path $plugins "westplay.dll")){throw "Westplay toujours présent : retrait nécessaire avant cet essai."}
  foreach($name in $entries) {
    $source=Join-Path $PSScriptRoot $name
    if(-not (Test-Path -LiteralPath $source)){throw "Fichier manquant dans EssieuPilot : $name"}
    if(Test-Path -LiteralPath (Join-Path $plugins $name)){throw "DLL déjà présente dans ATS : aucune modification ($name)"}
  }
  if((GetHash (Join-Path $PSScriptRoot "scs_telemetry_hub.dll")) -ne "357b97adafa8a3f9648e155fa4b07bba43a4d610d3e15b229d2cac102a817371"){throw "Télémétrie SHA256 invalide."}
  $created=New-Object System.Collections.Generic.List[string]
  try {
    if(-not (Test-Path -LiteralPath $plugins)){New-Item -ItemType Directory -Path $plugins | Out-Null}
    $records=@()
    foreach($name in $entries){
      $from=Join-Path $PSScriptRoot $name
      $to=Join-Path $plugins $name
      Copy-Item -LiteralPath $from -Destination $to -ErrorAction Stop
      $created.Add($to)
      if((GetHash $from) -ne (GetHash $to)){throw "Copie invalide : $name"}
      $records+=@{Name=$name;SHA256=(GetHash $from)}
    }
    @{Version=1;GamePath=$game;Files=$records} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $manifest -Encoding UTF8
    Write-Host "DLL installées. Ouvre EssieuPilot AVANT de lancer ATS 1.62."
  } catch {
    foreach($name in $entries){
      $to=Join-Path $plugins $name;$from=Join-Path $PSScriptRoot $name
      if($created.Contains($to) -and (Test-Path -LiteralPath $to) -and (GetHash $to) -eq (GetHash $from)){
        Remove-Item -LiteralPath $to -ErrorAction SilentlyContinue
      }
    }
    throw
  }
} catch {
  Write-Host ("Erreur : "+$_.Exception.Message) -ForegroundColor Red
  exit 1
}
