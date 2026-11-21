# EssieuPilot 1.6.3 RC4: visible dual status tile anchored near the original GUI.
# Uses a separate overlay process because the original 1.6.2 GUI source is unavailable.
# Green steering requires a recent acknowledgement written by the ATS DLL AFTER its hook executed.
param([int]$GuiPid = 0, [int]$ParentPid = 0, [switch]$SelfTest, [switch]$UiSmokeTest)
$ErrorActionPreference='Stop'
$diagnosticsDir=Join-Path $env:LOCALAPPDATA 'EssieuPilot\diagnostics'
New-Item -ItemType Directory -Force -Path $diagnosticsDir | Out-Null
$diagnosticsLog=Join-Path $diagnosticsDir 'indicateurs-1.6.3-rc4.log'
function Write-Diagnostic([string]$message) {
    try { Add-Content -LiteralPath $diagnosticsLog -Encoding UTF8 -Value (
        '[{0}] {1}' -f (Get-Date -Format o),$message) } catch { }
}
Write-Diagnostic ("RC4 watcher started; GuiPid=$GuiPid ParentPid=$ParentPid SelfTest=$SelfTest UiSmokeTest=$UiSmokeTest")
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class EPStatusNative {
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left,Top,Right,Bottom; }
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd,out RECT rect);
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hwnd,out uint processId);
    [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hwnd);
    [DllImport("kernel32.dll")] public static extern ulong GetTickCount64();
}
'@

function Get-AtsRunning {
    param([scriptblock]$Probe = { @(Get-Process -Name amtrucks -ErrorAction SilentlyContinue).Count -gt 0 })
    try {
        $r = & $Probe
        if ($r -isnot [bool]) { return $null }
        return $r
    } catch { return $null }
}

function Get-AppliedAck {
    $mmf=$null; $view=$null
    try {
        $mmf=[System.IO.MemoryMappedFiles.MemoryMappedFile]::OpenExisting(
            'Local\EssieuPilotAppliedV1',
            [System.IO.MemoryMappedFiles.MemoryMappedFileRights]::Read)
        $view=$mmf.CreateViewAccessor(0,32,[System.IO.MemoryMappedFiles.MemoryMappedFileAccess]::Read)
        $seq=$view.ReadUInt32(8)
        $magic=$view.ReadUInt32(0)
        $version=$view.ReadUInt32(4)
        $stamp=$view.ReadUInt64(16)
        $end=$view.ReadUInt32(12)
        $seqAgain=$view.ReadUInt32(8)
        if ($magic -ne 0x45504131 -or $version -ne 1 -or
            $seq -ne $end -or $seq -ne $seqAgain -or (($seq -band 1) -ne 0)) { return $false }
        $now=[EPStatusNative]::GetTickCount64()
        return ($now -ge $stamp -and ($now-$stamp) -le 1500)
    } catch {
        return $false
    } finally {
        if($null -ne $view){$view.Dispose()}
        if($null -ne $mmf){$mmf.Dispose()}
    }
}

if($SelfTest){
    if ((Get-AtsRunning -Probe { $true }) -ne $true) { throw 'ATS started test failed' }
    if ((Get-AtsRunning -Probe { $false }) -ne $false) { throw 'ATS stopped test failed' }
    if ($null -ne (Get-AtsRunning -Probe { throw 'probe error' })) { throw 'Unknown must not be stopped' }
    $testMap=[System.IO.MemoryMappedFiles.MemoryMappedFile]::CreateNew('Local\EssieuPilotAppliedV1',32)
    try {
        $v=$testMap.CreateViewAccessor()
        try {
            $v.Write(0,[uint32]0x45504131)
            $v.Write(4,[uint32]1)
            $v.Write(8,[uint32]2)
            $v.Write(12,[uint32]2)
            $v.Write(16,[uint64][EPStatusNative]::GetTickCount64())
            $v.Flush()
            if (-not (Get-AppliedAck)) { throw 'Fresh DLL acknowledgement not detected' }
            $v.Write(16,[uint64]([EPStatusNative]::GetTickCount64()-5000))
            $v.Flush()
            if (Get-AppliedAck) { throw 'Stale acknowledgement was accepted' }
        } finally { $v.Dispose() }
    } finally { $testMap.Dispose() }
    Write-Output 'PASS: ATS present/absent/unknown and steering ACK fresh/stale'
    exit 0
}

[System.Windows.Forms.Application]::EnableVisualStyles()
$panel=New-Object System.Windows.Forms.Form
$panel.Text='EssieuPilot - Connexions'
$panel.FormBorderStyle=[System.Windows.Forms.FormBorderStyle]::FixedSingle
$panel.ControlBox=$true
$panel.ShowInTaskbar=$true
$panel.StartPosition=[System.Windows.Forms.FormStartPosition]::Manual
$panel.TopMost=$false
$panel.BackColor=[System.Drawing.Color]::FromArgb(15,29,46)
$panel.ClientSize=New-Object System.Drawing.Size(330,88)
$panel.Location=New-Object System.Drawing.Point(40,120)
$panel.Font=New-Object System.Drawing.Font('Segoe UI',9)
$heading=New-Object System.Windows.Forms.Label
$heading.Text='ESSIEUPILOT - CONNEXIONS'
$heading.Font=New-Object System.Drawing.Font('Segoe UI',9,[System.Drawing.FontStyle]::Bold)
$heading.ForeColor=[System.Drawing.Color]::Turquoise
$heading.SetBounds(10,5,268,20)
$panel.Controls.Add($heading)
$gameLamp=New-Object System.Windows.Forms.Label
$gameLamp.Text=[char]0x25CF
$gameLamp.Font=New-Object System.Drawing.Font('Segoe UI',15,[System.Drawing.FontStyle]::Bold)
$gameLamp.SetBounds(10,25,24,25)
$panel.Controls.Add($gameLamp)
$gameText=New-Object System.Windows.Forms.Label
$gameText.ForeColor=[System.Drawing.Color]::White
$gameText.SetBounds(38,28,285,21)
$panel.Controls.Add($gameText)
$pilotLamp=New-Object System.Windows.Forms.Label
$pilotLamp.Text=[char]0x25CF
$pilotLamp.Font=New-Object System.Drawing.Font('Segoe UI',15,[System.Drawing.FontStyle]::Bold)
$pilotLamp.SetBounds(10,51,24,25)
$panel.Controls.Add($pilotLamp)
$pilotText=New-Object System.Windows.Forms.Label
$pilotText.ForeColor=[System.Drawing.Color]::White
$pilotText.SetBounds(38,54,285,21)
$panel.Controls.Add($pilotText)
$tooltip=New-Object System.Windows.Forms.ToolTip
$tooltip.SetToolTip($pilotText,'Vert = la DLL a execute un appel de direction recemment; roues non encore verifiees.')
$tooltip.SetToolTip($pilotLamp,'La commande doit etre acquittee par le module ATS dans le jeu.')
$script:gameDisplay='ATS : verification'
$script:pilotDisplay='Pilotage : non confirme'
$script:lastGame='initial'
$tray=New-Object System.Windows.Forms.NotifyIcon
$tray.Icon=[System.Drawing.SystemIcons]::Application
$tray.Text='EssieuPilot : controle ATS'
$tray.Visible=$true
$menu=New-Object System.Windows.Forms.ContextMenuStrip
$showItem=New-Object System.Windows.Forms.ToolStripMenuItem('Afficher les deux voyants')
$showItem.Add_Click({ $panel.Show(); $panel.WindowState=[System.Windows.Forms.FormWindowState]::Normal; $panel.Activate() })
[void]$menu.Items.Add($showItem)
$menuItem=New-Object System.Windows.Forms.ToolStripMenuItem('Afficher le diagnostic')
$menuItem.Add_Click({
    [void][System.Windows.Forms.MessageBox]::Show(
        "$script:gameDisplay" + [Environment]::NewLine + "$script:pilotDisplay" + [Environment]::NewLine +
        "Le vert du pilotage signifie retour de la DLL, pas verification du mouvement des roues.",
        'EssieuPilot', [System.Windows.Forms.MessageBoxButtons]::OK,
        [System.Windows.Forms.MessageBoxIcon]::Information)
})
[void]$menu.Items.Add($menuItem)
$tray.ContextMenuStrip=$menu
$tray.Add_DoubleClick({ [void]$showItem.PerformClick() })
$script:lastWarning=''
$script:closingByLauncher=$false
$panel.Add_FormClosing({
    if ($_.CloseReason -eq [System.Windows.Forms.CloseReason]::UserClosing -and
        -not $script:closingByLauncher) {
        $_.Cancel=$true
        $panel.Hide()
        Write-Diagnostic 'Panel manually hidden; recover using system tray.'
    }
})
$panel.Add_Shown({ Write-Diagnostic ("RC4 panel SHOWN; HWND=$($panel.Handle) visible=$($panel.Visible)") })
$timer=New-Object System.Windows.Forms.Timer
$timer.Interval=1000
$timer.Add_Tick({
    try {
        if ($ParentPid -gt 0 -and $null -eq (Get-Process -Id $ParentPid -ErrorAction SilentlyContinue)) {
            $script:closingByLauncher=$true
            Write-Diagnostic 'Parent launcher exited; closing status panel.'
            $panel.Close()
            return
        }
        $game=Get-AtsRunning
        $running=$null -ne $game -and $game -eq $true
        $script:gameDisplay=if($null -eq $game){'ATS : etat indisponible'}elseif($running){'ATS : EN LIGNE'}else{'ATS : HORS LIGNE'}
        $gameLamp.ForeColor=if($running){[System.Drawing.Color]::LimeGreen}else{[System.Drawing.Color]::OrangeRed}
        $gameText.Text=$script:gameDisplay
        $applied=$running -and (Get-AppliedAck)
        $script:pilotDisplay=if($applied){'Pilotage : retour DLL actif'}else{'Pilotage : non confirme'}
        $pilotLamp.ForeColor=if($applied){[System.Drawing.Color]::LimeGreen}else{[System.Drawing.Color]::OrangeRed}
        $pilotText.Text=$script:pilotDisplay
        $tray.Text=if($running){'EssieuPilot : ATS en ligne'}else{'EssieuPilot : ATS hors ligne'}
        if($script:lastGame -ne 'initial' -and $script:lastGame -ne $script:gameDisplay){
            $tray.ShowBalloonTip(2500,'EssieuPilot',$script:gameDisplay,[System.Windows.Forms.ToolTipIcon]::Info)
        }
        if ($script:lastGame -ne $script:gameDisplay) {Write-Diagnostic ($script:gameDisplay + '; ' + $script:pilotDisplay)}
        $script:lastGame=$script:gameDisplay
        # A previous RC3 bug silently hid the entire panel if the process had
        # no MainWindowHandle (bootstrap launcher) or focus was outside the GUI.
        # RC4 NEVER hides the panel because of a missing HWND.
        if ($GuiPid -gt 0) {
            $gui=Get-Process -Id $GuiPid -ErrorAction SilentlyContinue
            if ($null -ne $gui) {
                $gui.Refresh()
                $hwnd=$gui.MainWindowHandle
                if ($hwnd -ne [IntPtr]::Zero -and -not [EPStatusNative]::IsIconic($hwnd)) {
                    $rect=New-Object EPStatusNative+RECT
                    if ([EPStatusNative]::GetWindowRect($hwnd,[ref]$rect)) {
                        $screen=[System.Windows.Forms.Screen]::FromHandle($hwnd).WorkingArea
                        $panel.Left=[Math]::Max($screen.Left,[Math]::Min(
                            $rect.Right-$panel.Width-24,$screen.Right-$panel.Width))
                        $panel.Top=[Math]::Max($screen.Top,[Math]::Min(
                            $rect.Top+95,$screen.Bottom-$panel.Height))
                    }
                } elseif ($script:lastWarning -ne 'no_handle') {
                    Write-Diagnostic 'GUI has no window handle yet; keeping the status panel visible separately.'
                    $script:lastWarning='no_handle'
                }
            }
        }
        # Only float above the original GUI while it is foreground.
        # Do not cover ATS when the player switches into the game.
        $fg=[EPStatusNative]::GetForegroundWindow()
        [uint32]$fgpid=0
        if ($fg -ne [IntPtr]::Zero) {
            [void][EPStatusNative]::GetWindowThreadProcessId($fg,[ref]$fgpid)
        }
        $panel.TopMost=($fgpid -eq [uint32]$GuiPid -or $fgpid -eq [uint32]$PID)
    } catch {
        # Log rather than silently hiding the only visible status indicator.
        $message=$_.Exception.ToString()
        if ($message -ne $script:lastWarning) {
            Write-Diagnostic ("Status panel tick error: " + $message)
            $script:lastWarning=$message
        }
        $panel.TopMost=$false
    }
})
if($UiSmokeTest){
    $panel.Add_Shown({
        if (-not $panel.Visible -or $panel.Handle -eq [IntPtr]::Zero -or -not $panel.ShowInTaskbar) {
            Write-Diagnostic 'UI smoke test failed: form invisible or no handle/taskbar presence.'
            [Environment]::Exit(3)
        }
        Write-Diagnostic 'UI smoke test passed: form visible and has a taskbar entry.'
        $script:closingByLauncher=$true
        $panel.Close()
    })
}
try{
    $timer.Start()
    # Application.Run(Form) guarantees a first Show/Shown event; the old
    # Application.Run(ApplicationContext) never opened the form at startup.
    [System.Windows.Forms.Application]::Run($panel)
} catch {
    Write-Diagnostic ("Watcher fatal: " + $_.Exception.ToString())
    throw
} finally{
    Write-Diagnostic 'Watcher exited.'
    $timer.Stop()
    $timer.Dispose()
    $tray.Visible=$false
    $tray.Dispose()
    $panel.Dispose()
    $menu.Dispose()
    $tooltip.Dispose()
}
