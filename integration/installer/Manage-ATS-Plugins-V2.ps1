# EssieuPilot managed ATS modules — transactional, repeatable installer.
# Works for first-time installs and future in-app updates, without asking
# users to copy files manually. Not an authorization to distribute an
# ATS 1.62-unvalidated steering DLL.
param(
    [ValidateSet("Preflight","Install","Remove","Status")][string]$Action="Status",
    [string]$GamePath="",
    [string]$PackageDir=$PSScriptRoot,
    [string]$StateDir=""
)
Set-StrictMode -Version Latest
$ErrorActionPreference="Stop"
$names=@("EssieuPilotATS.dll","scs_telemetry_hub.dll")
if (-not $StateDir) {
    if (-not $env:LOCALAPPDATA) {throw "LOCALAPPDATA indisponible"}
    $StateDir=Join-Path $env:LOCALAPPDATA "EssieuPilot\ATS-Modules"
}
$recordPath=Join-Path $StateDir "installed.json"
function Digest([string]$f) {
    (Get-FileHash -Algorithm SHA256 -LiteralPath $f).Hash.ToLowerInvariant()
}
function ATSClosed {
    if (Get-Process -Name "amtrucks" -ErrorAction SilentlyContinue) {
        throw "ATS est lancé. Ferme le jeu avant la mise à jour des DLL."
    }
}
function GameRoot([string]$given) {
    $paths=New-Object 'System.Collections.Generic.List[string]'
    if ($given) {$paths.Add($given)}
    else {
        $steamRoots=@()
        try {
            $steamProps=Get-ItemProperty "HKCU:\Software\Valve\Steam" -ErrorAction Stop
            if ($steamProps.SteamPath) {$steamRoots+=([string]$steamProps.SteamPath).Replace("/","\")}
        } catch {}
        $steamRoots+=@(
            "$env:ProgramFiles(x86)\Steam",
            "$env:ProgramFiles\Steam",
            "C:\Steam", "D:\Steam", "C:\SteamLibrary", "D:\SteamLibrary",
            "E:\SteamLibrary", "F:\SteamLibrary", "G:\SteamLibrary"
        )
        foreach ($steam in ($steamRoots|Where-Object { $_ }|Select-Object -Unique)) {
            $paths.Add((Join-Path $steam "steamapps\common\American Truck Simulator"))
            $vdf=Join-Path $steam "steamapps\libraryfolders.vdf"
            if (Test-Path -LiteralPath $vdf -PathType Leaf) {
                $txt=Get-Content -LiteralPath $vdf -Raw
                foreach($m in [regex]::Matches($txt,'(?m)^\s*"path"\s*"([^"]+)"')){
                    $lib=$m.Groups[1].Value.Replace("\\","\")
                    $paths.Add((Join-Path $lib "steamapps\common\American Truck Simulator"))
                }
            }
        }
    }
    $found=New-Object 'System.Collections.Generic.List[string]'
    foreach($p in $paths) {
        if (-not $p) {continue}
        $exe=Join-Path $p "bin\win_x64\amtrucks.exe"
        if (Test-Path -LiteralPath $exe -PathType Leaf) {
            $resolved=(Resolve-Path -LiteralPath $p).Path
            $dir=Get-Item -LiteralPath $p -Force
            if (($dir.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                throw "Dossier ATS non standard : lien symbolique refusé."
            }
            if (-not $found.Contains($resolved)) {$found.Add($resolved)}
        }
    }
    if ($found.Count -eq 0) {throw "ATS introuvable. L'application peut être installée sans DLL ; indique le dossier ATS pour les modules."}
    if ($found.Count -gt 1) {throw "Plusieurs installations ATS détectées : sélectionne le dossier cible."}
    return $found[0]
}
function LoadPackage {
    $path=Join-Path $PackageDir "module-manifest.json"
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Paquet DLL validé manquant : aucun module installé dans ATS."
    }
    $m=Get-Content -LiteralPath $path -Raw|ConvertFrom-Json
    if ($m.schema -ne 1 -or $m.game -ne "ats" -or
        $m.game_series -ne "1.62" -or
        [string]::IsNullOrWhiteSpace([string]$m.module_version) -or
        @($m.files).Count -ne 2) {throw "Manifeste de modules invalide."}
    foreach($name in $names){
        $items=@($m.files|Where-Object {$_.name -ceq $name})
        if($items.Count -ne 1){throw "DLL absente ou doublée dans le manifeste : $name"}
        $item=$items[0]
        if(([string]$item.sha256) -cnotmatch '^[0-9a-f]{64}$') {throw "SHA256 absent : $name"}
        $from=Join-Path $PackageDir $name
        if (-not (Test-Path -LiteralPath $from -PathType Leaf) -or
            (Digest $from) -cne $item.sha256) {
            throw "DLL fournie invalide ou absente : $name"
        }
    }
    return $m
}
function OldRecord {
    if (-not (Test-Path -LiteralPath $recordPath -PathType Leaf)){return $null}
    $r=Get-Content -LiteralPath $recordPath -Raw|ConvertFrom-Json
    if ($r.schema -ne 1 -or @($r.files).Count -ne 2 -or
        [string]::IsNullOrWhiteSpace([string]$r.game_path)){throw "Enregistrement DLL illisible."}
    foreach($name in $names) {
        $items=@($r.files|Where-Object {$_.name -ceq $name})
        if($items.Count -ne 1 -or [string]$items[0].sha256 -cnotmatch '^[0-9a-f]{64}$') {
            throw "Enregistrement DLL invalide : $name"
        }
    }
    return $r
}
function CheckTarget([string]$plugins,$previous) {
    foreach($name in $names) {
        $to=Join-Path $plugins $name
        if (-not (Test-Path -LiteralPath $to)){continue}
        $file=Get-Item -LiteralPath $to -Force
        if($file.PSIsContainer -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint)){
            throw "Fichier de plugin non standard : $name"
        }
        if ($null -eq $previous) {
            throw "DLL déjà présente et non gérée par EssieuPilot : $name. Aucune modification."
        }
        $entry=@($previous.files|Where-Object {$_.name -ceq $name})[0]
        if ((Digest $to) -cne [string]$entry.sha256) {
            throw "DLL existante modifiée par un tiers : $name. Refus du remplacement."
        }
    }
}
try {
    $previous=OldRecord
    if ($previous -and -not $GamePath) {$GamePath=[string]$previous.game_path}
    $root=GameRoot $GamePath
    if($previous -and [string]$previous.game_path -cne $root) {
        throw "L'installation enregistrée appartient à un autre dossier ATS."
    }
    $plugins=Join-Path $root "bin\win_x64\plugins"
    if(Test-Path -LiteralPath $plugins) {
        $dir=Get-Item -LiteralPath $plugins -Force
        if (-not $dir.PSIsContainer -or ($dir.Attributes -band [IO.FileAttributes]::ReparsePoint)){
            throw "Dossier plugins ATS non standard : aucune modification."
        }
    }
    if ($Action -eq "Status"){
        Write-Host "ATS: $root"
        Write-Host "EssieuPilot managed: $($null -ne $previous)"
        foreach($name in $names){Write-Host "$name : $(Test-Path -LiteralPath (Join-Path $plugins $name))"}
        exit 0
    }
    ATSClosed
    if ($Action -eq "Remove") {
        if ($null -eq $previous) {Write-Host "Aucun module EssieuPilot enregistré.";exit 0}
        CheckTarget $plugins $previous
        foreach($name in $names) {
            $to=Join-Path $plugins $name
            if(Test-Path -LiteralPath $to){Remove-Item -LiteralPath $to -ErrorAction Stop}
        }
        Remove-Item -LiteralPath $recordPath -ErrorAction Stop
        Write-Host "DLL EssieuPilot retirées. Les autres plugins ATS sont conservés."
        exit 0
    }
    $package=LoadPackage
    CheckTarget $plugins $previous
    if($Action -eq "Preflight"){Write-Host "Modules validés, ATS fermé, emplacement prêt : $root";exit 0}
    if(-not(Test-Path -LiteralPath $plugins)){New-Item -ItemType Directory -Path $plugins -Force|Out-Null}
    New-Item -ItemType Directory -Path $StateDir -Force|Out-Null
    # Backups of only managed DLLs; never overwrite third-party files.
    $backup=Join-Path $StateDir ("rollback-"+[guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $backup -Force|Out-Null
    $had=@{}
    $oldManifest=(Test-Path -LiteralPath $recordPath)
    try {
        if ($oldManifest){Copy-Item -LiteralPath $recordPath -Destination (Join-Path $backup "installed.json")}
        foreach($name in $names) {
            $to=Join-Path $plugins $name
            $had[$name]=Test-Path -LiteralPath $to
            if($had[$name]){Copy-Item -LiteralPath $to -Destination (Join-Path $backup $name)}
        }
        foreach($name in $names) {
            $from=Join-Path $PackageDir $name
            $to=Join-Path $plugins $name
            Copy-Item -LiteralPath $from -Destination $to -Force -ErrorAction Stop
            if ((Digest $to) -cne (Digest $from)){throw "Copie DLL invalide : $name"}
        }
        $records=@($package.files|ForEach-Object {
            @{name=[string]$_.name;sha256=[string]$_.sha256}
        })
        $new=@{schema=1;game_path=$root;module_version=[string]$package.module_version;files=$records}
        $tmp=Join-Path $StateDir ("installed-"+[guid]::NewGuid().ToString("N")+".json")
        $new|ConvertTo-Json -Depth 4|Set-Content -LiteralPath $tmp -Encoding UTF8
        Move-Item -LiteralPath $tmp -Destination $recordPath -Force -ErrorAction Stop
        Write-Host "EssieuPilot et ses DLL synchronisés : $($package.module_version)"
    } catch {
        $failure=$_
        foreach($name in $names){
            $to=Join-Path $plugins $name
            $from=Join-Path $backup $name
            if(Test-Path -LiteralPath $from){Copy-Item -LiteralPath $from -Destination $to -Force}
            elseif ($had.ContainsKey($name) -and -not $had[$name] -and (Test-Path -LiteralPath $to)){
                Remove-Item -LiteralPath $to -Force
            }
        }
        if(Test-Path -LiteralPath (Join-Path $backup "installed.json")){
            Copy-Item -LiteralPath (Join-Path $backup "installed.json") -Destination $recordPath -Force
        }
        throw "Mise à jour DLL annulée et fichiers précédents restaurés : $($failure.Exception.Message)"
    } finally {
        if(Test-Path -LiteralPath $backup){Remove-Item -LiteralPath $backup -Recurse -Force -ErrorAction SilentlyContinue}
    }
} catch {
    Write-Error ("EssieuPilot modules ATS : "+$_.Exception.Message) -ErrorAction Continue
    exit 1
}
