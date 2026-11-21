# EssieuPilot Original.11 : installation/retrait gérés de la DLL expérimentale.
# Jamais de remplacement d'une DLL tiers ou mod non possédé par EssieuPilot.
param(
 [ValidateSet('Install','Remove','Status')][string]$Action='Status',
 [string]$GamePath='', [string]$PackageDir=$PSScriptRoot,
 [string]$StateDir='', [string]$ConsentPath='',
 [switch]$AllowPhysicalTest, [switch]$TestSkipGameHash
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$expected='63ecae85a070934530d9fd971d4d56e5479f1214d7077b403f79b8981c613e08'
$name='EssieuPilotATS.dll'
# Public Original.10 standalone DLL, published Nov 21, 2026. Only this
# known old SHA is eligible for a safe one-time manual-to-managed migration.
$knownManualOriginal10='a43a25738e0f07e0d58ca1331d009412490a146863ac8366d9baded7d0e837c3'
if(!$StateDir){$StateDir=Join-Path $env:LOCALAPPDATA 'EssieuPilot\ATS-Physical-Test'}
if(!$ConsentPath){$ConsentPath=Join-Path $env:APPDATA 'EssieuPilot-OriginalTest\physical-test-consent.txt'}
$record=Join-Path $StateDir 'installed.json'
function sha($path){ (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() }
function game($given){
 if($given){
  $path=[IO.Path]::GetFullPath($given)
  if(Test-Path -LiteralPath $path -PathType Leaf){
   if([IO.Path]::GetFileName($path) -ieq 'amtrucks.exe'){
    $path=Split-Path (Split-Path (Split-Path $path -Parent) -Parent) -Parent
   }
  }
  if(-not(Test-Path -LiteralPath (Join-Path $path 'bin\win_x64\amtrucks.exe') -PathType Leaf)){
   throw "Dossier ATS invalide : $path"
  }
  return $path
 }
 # Original.10 did not search for Steam installed on D:\Program Files (x86).
 # The actual ATS game.log shows this path; search the normal Steam registry
 # as well as C..G drive Program Files paths and Steam libraryfolders.vdf.
 $roots=New-Object 'System.Collections.Generic.List[string]'
 foreach($regKey in @('HKCU:\Software\Valve\Steam','HKLM:\SOFTWARE\WOW6432Node\Valve\Steam',
   'HKLM:\SOFTWARE\Valve\Steam')){
  try{
   $reg=Get-ItemProperty -LiteralPath $regKey -ErrorAction Stop
   foreach($prop in @('SteamPath','InstallPath')){
    if($reg.$prop){$roots.Add(([string]$reg.$prop).Replace('/','\'))}
   }
  }catch{}
 }
 foreach($drive in @('C:','D:','E:','F:','G:')){
  foreach($subdir in @('Program Files (x86)\Steam','Program Files\Steam',
    'Steam','SteamLibrary','Games\Steam','Games\SteamLibrary')){
   $roots.Add($drive+'\'+$subdir) # Do not Join-Path before confirming the PSDrive exists
  }
 }
 $possible=New-Object 'System.Collections.Generic.List[string]'
 foreach($r in ($roots|Where-Object {$_ -and [IO.Directory]::Exists($_)}|Select-Object -Unique)){
  $possible.Add((Join-Path $r 'steamapps\common\American Truck Simulator'))
  $vdf=Join-Path $r 'steamapps\libraryfolders.vdf'
  if([IO.File]::Exists($vdf)){
   foreach($m in [regex]::Matches((Get-Content -LiteralPath $vdf -Raw),'"path"\s+"([^"]+)"')){
    $lib=$m.Groups[1].Value.Replace('\\','\')
    $possible.Add((Join-Path $lib 'steamapps\common\American Truck Simulator'))
   }
  }
 }
 $found=@($possible|Where-Object {
  [IO.File]::Exists((Join-Path $_ 'bin\win_x64\amtrucks.exe'))
 }|ForEach-Object {[IO.Path]::GetFullPath($_)}|Select-Object -Unique)
 if($found.Count -eq 1){return $found[0]}
 if($found.Count -gt 1){
  # Disambiguate safely by the only executable hash authorized for this DLL.
  $matching=@($found|Where-Object {(sha (Join-Path $_ 'bin\win_x64\amtrucks.exe')) -ceq $expected})
  if($matching.Count -eq 1){return $matching[0]}
 }
 throw "ATS introuvable ou plusieurs installations : vérifie le dossier Steam et sélectionne-le dans l'installateur. Détectés : $($found.Count)"
}
function oldRecord {
 if(-not(Test-Path -LiteralPath $record)){return $null}
 $r=Get-Content -LiteralPath $record -Raw|ConvertFrom-Json
 if($r.schema -ne 1 -or $r.name -cne $name -or
    [string]$r.sha256 -cnotmatch '^[0-9a-f]{64}$' -or
    -not $r.game_path){throw 'Enregistrement de DLL invalide : aucune modification'}
 return $r
}
try{
 $old=oldRecord
 if($old -and !$GamePath){$GamePath=[string]$old.game_path}
 $root=game $GamePath
 if($old -and ([IO.Path]::GetFullPath([string]$old.game_path) -ine [IO.Path]::GetFullPath($root))){throw 'Installation ATS différente du registre'}
 $exe=Join-Path $root 'bin\win_x64\amtrucks.exe'
 $plugindir=Join-Path $root 'bin\win_x64\plugins'
 $target=Join-Path $plugindir $name
 if($Action -eq 'Install' -and -not $TestSkipGameHash -and (sha $exe) -cne $expected){throw 'ATS 1.62.2.2 non reconnu : empreinte différente, DLL non installée'}
 if(Test-Path -LiteralPath $plugindir){
  $dir=Get-Item -LiteralPath $plugindir -Force
  if(-not $dir.PSIsContainer -or ($dir.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'Dossier plugins non régulier'}
 }
 if($Action -eq 'Status'){
  Write-Host "ATS : $root"
  Write-Host "DLL présente : $(Test-Path -LiteralPath $target)"
  Write-Host "Gérée par EssieuPilot : $($null -ne $old)"
  Write-Host "Consentement physique : $(Test-Path -LiteralPath $ConsentPath)"
  exit 0
 }
 if(Get-Process -Name 'amtrucks' -ErrorAction SilentlyContinue){throw 'Ferme ATS avant installation ou retrait de la DLL'}
 if(Test-Path -LiteralPath $target){
  $file=Get-Item -LiteralPath $target -Force
  if($file.PSIsContainer -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint)){throw 'Fichier cible non régulier'}
  $currentHash=sha $target
  $knownManualMigration=($Action -eq 'Install' -and $null -eq $old -and
    $currentHash -ceq $knownManualOriginal10)
  if(-not $knownManualMigration -and
     ($null -eq $old -or $currentHash -cne [string]$old.sha256)){
    throw 'DLL existante non gérée/modifiée : aucune écriture autorisée'
  }
  if($knownManualMigration){Write-Host 'Migration sécurisée : DLL Original.10 manuelle reconnue par son SHA-256'}
 }
 if($Action -eq 'Remove'){
  if($old){
   if(Test-Path -LiteralPath $target){Remove-Item -LiteralPath $target -Force}
   Remove-Item -LiteralPath $record -Force
  }
  if(Test-Path -LiteralPath $ConsentPath){Remove-Item -LiteralPath $ConsentPath -Force}
  Write-Host 'DLL EssieuPilot de test retirée ; autres plugins inchangés'
  exit 0
 }
 $src=Join-Path $PackageDir $name
 $manifest=Join-Path $PackageDir 'module-manifest.json'
 if(-not(Test-Path -LiteralPath $src -PathType Leaf) -or
    -not(Test-Path -LiteralPath $manifest -PathType Leaf)){throw 'DLL ou manifeste absent'}
 $m=Get-Content -LiteralPath $manifest -Raw|ConvertFrom-Json
 if($m.schema -ne 1 -or $m.game -ne 'ats' -or $m.game_series -ne '1.62' -or
    @($m.files).Count -ne 1 -or [string]$m.files[0].name -cne $name -or
    [string]$m.files[0].sha256 -cnotmatch '^[0-9a-f]{64}$' -or
    [string]$m.module_version -notmatch 'original\.11' -or
    (sha $src) -cne [string]$m.files[0].sha256){throw 'Paquet DLL invalide ou non vérifié SHA-256'}
 New-Item -ItemType Directory -Path $plugindir,$StateDir -Force|Out-Null
 $backup=Join-Path $StateDir ('rollback-'+[guid]::NewGuid().ToString('N'))
 New-Item -ItemType Directory -Path $backup -Force|Out-Null
 $had=Test-Path -LiteralPath $target
 if($had){Copy-Item -LiteralPath $target (Join-Path $backup $name)}
 if(Test-Path -LiteralPath $record){Copy-Item -LiteralPath $record (Join-Path $backup 'installed.json')}
 try{
  Copy-Item -LiteralPath $src -Destination $target -Force
  if((sha $target) -cne [string]$m.files[0].sha256){throw 'Copie DLL corrompue'}
  @{schema=1;name=$name;sha256=[string]$m.files[0].sha256;game_path=$root;module_version=[string]$m.module_version}|
   ConvertTo-Json|Set-Content -LiteralPath $record -Encoding UTF8
  if($AllowPhysicalTest){
   New-Item -ItemType Directory -Path (Split-Path $ConsentPath -Parent) -Force|Out-Null
   Set-Content -LiteralPath $ConsentPath -Value 'I_ACCEPT_CRASH_RISK' -Encoding ASCII
  }elseif(Test-Path -LiteralPath $ConsentPath){Remove-Item -LiteralPath $ConsentPath -Force}
  $previousError=Join-Path $StateDir 'dernier-echec-installation.txt'
  if(Test-Path -LiteralPath $previousError){Remove-Item -LiteralPath $previousError -Force}
  Write-Host "PASS : DLL installée, empreinte vérifiée, destination : $target"
  Write-Host "Essai physique activé (solo seulement) : $AllowPhysicalTest"
 }catch{
  if($had){Copy-Item -LiteralPath (Join-Path $backup $name) -Destination $target -Force}
  elseif(Test-Path -LiteralPath $target){Remove-Item -LiteralPath $target -Force}
  $saved=Join-Path $backup 'installed.json'
  if(Test-Path -LiteralPath $saved){Copy-Item -LiteralPath $saved -Destination $record -Force}
  elseif(Test-Path -LiteralPath $record){Remove-Item -LiteralPath $record -Force}
  throw
 }
 exit 0
}catch{
 $message='EssieuPilot : '+$_.Exception.Message
 try {
  New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
  # Persist a terse failure report for post-install diagnostics.
  Set-Content -LiteralPath (Join-Path $StateDir 'dernier-echec-installation.txt') -Value $message -Encoding UTF8
 }catch{}
 Write-Error $message
 exit 1
}
