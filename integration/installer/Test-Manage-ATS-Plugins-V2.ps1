# Windows CI regression tests using only FAKE DLL files and a fake ATS root.
# No real game files are ever modified.
$ErrorActionPreference="Stop"
Set-StrictMode -Version Latest
$root=Join-Path $env:RUNNER_TEMP ("EssieuPilot-V2-Modules-"+[guid]::NewGuid().ToString("N"))
$game=Join-Path $root "American Truck Simulator"
$plugins=Join-Path $game "bin\win_x64\plugins"
$package=Join-Path $root "package"
$state=Join-Path $root "state"
$manager=Join-Path $PSScriptRoot "Manage-ATS-Plugins-V2.ps1"
$names=@("EssieuPilotATS.dll","scs_telemetry_hub.dll")
function Fail($message){throw "DLL V2 TEST: $message"}
function Hash([string]$p){(Get-FileHash -LiteralPath $p -Algorithm SHA256).Hash.ToLowerInvariant()}
function Package([string]$version){
    New-Item -ItemType Directory -Path $package -Force|Out-Null
    $files=@()
    foreach($name in $names){
        [IO.File]::WriteAllBytes((Join-Path $package $name),
            [Text.Encoding]::UTF8.GetBytes("FAKE TEST ONLY : $version : $name"))
        $files+=@{name=$name;sha256=Hash(Join-Path $package $name)}
    }
    @{schema=1;game="ats";game_series="1.62";module_version=$version;files=$files} |
        ConvertTo-Json -Depth 4|Set-Content -LiteralPath (Join-Path $package "module-manifest.json") -Encoding UTF8
}
function Run([string]$action,[bool]$expectSuccess=$true){
    & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $manager -Action $action -GamePath $game -PackageDir $package -StateDir $state
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
}
try {
    New-Item -ItemType Directory -Path $plugins -Force|Out-Null
    [IO.File]::WriteAllBytes((Join-Path $game "bin\win_x64\amtrucks.exe"),
        [Text.Encoding]::UTF8.GetBytes("FAKE ATS TEST ONLY"))
    [IO.File]::WriteAllBytes((Join-Path $plugins "westplay.dll"),
        [Text.Encoding]::UTF8.GetBytes("UNRELATED PLUGIN - KEEP"))
    $unrelatedHash=Hash (Join-Path $plugins "westplay.dll")
    Package "test-v1"
    Run "Preflight"
    Run "Install"
    AssertVersion "test-v1"
    Run "Install" # idempotent, safe reinstall
    AssertVersion "test-v1"
    Package "test-v2"
    Run "Preflight"
    Run "Install"
    AssertVersion "test-v2" # previously managed files updated in-place
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
    if((Hash (Join-Path $plugins "westplay.dll")) -ne $unrelatedHash){Fail "third-party DLL was modified"}
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
    Write-Host "PASS: first install, update, reinstall, remove, hashes, third-party preservation, collision and tampering refusal."
} finally {
    if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue}
}
