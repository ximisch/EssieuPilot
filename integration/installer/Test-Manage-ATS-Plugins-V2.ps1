# Windows CI regression tests using only FAKE DLL files and a fake ATS root.
# No real game files are ever modified.
$ErrorActionPreference="Stop"
Set-StrictMode -Version Latest
$root=Join-Path $env:RUNNER_TEMP ("EssieuPilot-V2-Modules-"+[guid]::NewGuid().ToString("N"))
$game=Join-Path $root "American Truck Simulator"
$plugins=Join-Path $game "bin\win_x64\plugins"
$mods=Join-Path $root "Documents\American Truck Simulator\mod"
$package=Join-Path $root "package"
$state=Join-Path $root "state"
$manager=Join-Path $PSScriptRoot "Manage-ATS-Plugins-V2.ps1"
$names=@("EssieuPilotATS.dll","scs_telemetry_hub.dll")
function Fail($message){throw "DLL V2 TEST: $message"}
function Hash([string]$p){(Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Package([string]$version,[string[]]$scs=@()) {
    New-Item -ItemType Directory -Path $package -Force|Out-Null
    $files=@()
    foreach($name in $names){
        [IO.File]::WriteAllBytes((Join-Path $package $name),
            [Text.Encoding]::UTF8.GetBytes("FAKE TEST ONLY : $version : $name"))
        $files+=@{name=$name;sha256=Hash(Join-Path $package $name)}
    }
    Get-ChildItem -LiteralPath $package -Filter "*.scs" -File -ErrorAction SilentlyContinue |
        Remove-Item -Force
    $modRecords=@()
    foreach($name in $scs) {
        $file=Join-Path $package $name
        [IO.File]::WriteAllBytes($file,[Text.Encoding]::UTF8.GetBytes("FAKE .SCS - $version - $name"))
        $modRecords+=@{name=$name;sha256=Hash $file}
    }
    @{schema=1;game="ats";game_series="1.62";module_version=$version;files=$files;mods=$modRecords} |
        ConvertTo-Json -Depth 4|Set-Content -LiteralPath (Join-Path $package "module-manifest.json") -Encoding UTF8
}
function Run([string]$action,[bool]$expectSuccess=$true){
    & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $manager -Action $action -GamePath $game -PackageDir $package -StateDir $state -ModPath $mods
    $exit=$LASTEXITCODE
    if($expectSuccess -and $exit -ne 0){Fail "$action failed (code $exit)"}
    if(-not $expectSuccess -and $exit -eq 0){Fail "$action unexpectedly accepted invalid or foreign DLL"}
}
function AssertVersion([string]$version){
    $r=Get-Content -LiteralPath (Join-Path $state "installed.json") -Raw|ConvertFrom-Json
    if($r.module_version -cne $version){Fail "manifest version mismatch $($r.module_version)"}
    foreach($name in $names) {
        if((Hash (Join-Path $plugins $name)) -ne (Hash (Join-Path $package $name))){Fail "DLL not synced: $name"}
    }
    $p=Get-Content -LiteralPath (Join-Path $package "module-manifest.json") -Raw|ConvertFrom-Json
    foreach($item in $p.mods){
        if((Hash (Join-Path $mods $item.name)) -ne (Hash (Join-Path $package $item.name))){
            Fail "mod .scs not synced: $($item.name)"
        }
    }
}
try {
    New-Item -ItemType Directory -Path $plugins,$mods -Force|Out-Null
    [IO.File]::WriteAllBytes((Join-Path $game "bin\win_x64\amtrucks.exe"),
        [Text.Encoding]::UTF8.GetBytes("FAKE ATS TEST ONLY"))
    [IO.File]::WriteAllBytes((Join-Path $plugins "westplay.dll"),
        [Text.Encoding]::UTF8.GetBytes("UNRELATED PLUGIN - KEEP"))
    $unrelatedHash=Hash (Join-Path $plugins "westplay.dll")
    Set-Content -LiteralPath (Join-Path $mods "OtherCreator.scs") -Value "KEEP THIRD PARTY"
    $thirdPartyMod=Hash (Join-Path $mods "OtherCreator.scs")
    Package "test-v1" @("EssieuPilot_Steering.scs")
    Run "Preflight"
    Run "Install"
    AssertVersion "test-v1"
    Run "Install" # idempotent, safe reinstall
    AssertVersion "test-v1"
    Package "test-v2" @("EssieuPilot_Steering.scs","EssieuPilot_Axles.scs")
    Run "Preflight"
    Run "Install"
    AssertVersion "test-v2" # previously managed DLL and .scs files updated in-place
    Package "test-v3" @("EssieuPilot_Axles.scs")
    Run "Install"
    AssertVersion "test-v3"
    if(Test-Path -LiteralPath (Join-Path $mods "EssieuPilot_Steering.scs")){
        Fail "obsolete owned .scs mod not removed during upgrade"
    }
    # Tampered game DLL must never be replaced by update.
    Set-Content -LiteralPath (Join-Path $plugins $names[0]) -Value "UNRECOGNIZED GAME DLL"
    Run "Install" $false
    if((Get-Content -Raw -LiteralPath (Join-Path $plugins $names[0])).Trim() -ne "UNRECOGNIZED GAME DLL"){
        Fail "tampered DLL was overwritten"
    }
    Run "Remove" $false
    # Restore known managed version, then remove only owned files.
    Copy-Item -LiteralPath (Join-Path $package $names[0]) -Destination (Join-Path $plugins $names[0]) -Force
    Run "Remove"
    foreach($name in $names) {
        if(Test-Path -LiteralPath (Join-Path $plugins $name)){Fail "$name not removed"}
    }
    foreach($name in @("EssieuPilot_Steering.scs","EssieuPilot_Axles.scs")){
        if(Test-Path -LiteralPath (Join-Path $mods $name)){Fail "$name not removed"}
    }
    if((Hash (Join-Path $plugins "westplay.dll")) -ne $unrelatedHash){Fail "third-party DLL was modified"}
    if((Hash (Join-Path $mods "OtherCreator.scs")) -ne $thirdPartyMod){Fail "third-party .scs mod was modified"}
    # An unowned mod with EssieuPilot's name must never be overwritten.
    Set-Content -LiteralPath (Join-Path $mods "EssieuPilot_Axles.scs") -Value "FOREIGN .SCS"
    Run "Install" $false
    if((Get-Content -LiteralPath (Join-Path $mods "EssieuPilot_Axles.scs") -Raw).Trim() -ne "FOREIGN .SCS"){
        Fail "foreign .scs mod was overwritten"
    }
    Remove-Item -LiteralPath (Join-Path $mods "EssieuPilot_Axles.scs")
    # Existing unowned file with same name is never touched.
    Set-Content -LiteralPath (Join-Path $plugins $names[0]) -Value "FOREIGN OWNER"
    Run "Install" $false
    if((Get-Content -Raw -LiteralPath (Join-Path $plugins $names[0])).Trim() -ne "FOREIGN OWNER"){
        Fail "foreign DLL collision overwritten"
    }
    Remove-Item -LiteralPath (Join-Path $plugins $names[0])
    # Source digest mismatch is refused before installing any DLL.
    Set-Content -LiteralPath (Join-Path $package $names[0]) -Value "BAD HASH"
    Run "Preflight" $false
    if(Test-Path -LiteralPath (Join-Path $plugins $names[1])){Fail "unvalidated source reached ATS"}
    # Bad or unsafe SCS package names are rejected before installation.
    Package "test-v4" @("EssieuPilot_Valid.scs")
    $p=Join-Path $package "module-manifest.json"
    $data=Get-Content -LiteralPath $p -Raw|ConvertFrom-Json
    $data.mods[0].name="../GameData.scs"
    $data|ConvertTo-Json -Depth 5|Set-Content -LiteralPath $p
    Run "Preflight" $false
    if(Test-Path -LiteralPath (Join-Path $mods "EssieuPilot_Valid.scs")){Fail "unsafe SCS manifest accepted"}
    Write-Host "PASS: first install, DLL + .scs update, stale SCS removal, reinstall, remove, hashes, third-party preservation, collision and tampering refusal."
} finally {
    if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue}
}

$global:LASTEXITCODE=0
