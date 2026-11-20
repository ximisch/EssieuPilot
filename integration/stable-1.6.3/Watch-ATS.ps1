# EssieuPilot 1.6.3 RC3: in-application ATS/steering status panel.
# Uses a separate overlay process because the original 1.6.2 GUI source is unavailable.
# Green steering requires a recent acknowledgement written by the ATS DLL AFTER its hook executed.
param([int]$GuiPid = 0, [int]$ParentPid = 0, [switch]$SelfTest)
$ErrorActionPreference='Stop'
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
$context=New-Object System.Windows.Forms.ApplicationContext
$panel=New-Object System.Windows.Forms.Form
$panel.Text='EssieuPilot - Connexions'
$panel.FormBorderStyle=[System.Windows.Forms.FormBorderStyle]::FixedSingle
$panel.ControlBox=$false
$panel.ShowInTaskbar=$false
$panel.StartPosition=[System.Windows.Forms.FormStartPosition]::Manual
$panel.TopMost=$true
$panel.BackColor=[System.Drawing.Color]::FromArgb(15,29,46)
$panel.ClientSize=New-Object System.Drawing.Size(290,82)
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
$gameText.SetBounds(38,28,246,21)
$panel.Controls.Add($gameText)
$pilotLamp=New-Object System.Windows.Forms.Label
$pilotLamp.Text=[char]0x25CF
$pilotLamp.Font=New-Object System.Drawing.Font('Segoe UI',15,[System.Drawing.FontStyle]::Bold)
$pilotLamp.SetBounds(10,51,24,25)
$panel.Controls.Add($pilotLamp)
$pilotText=New-Object System.Windows.Forms.Label
$pilotText.ForeColor=[System.Drawing.Color]::White
$pilotText.SetBounds(38,54,246,21)
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
$tray.Add_DoubleClick({ [void]$menuItem.PerformClick() })
$timer=New-Object System.Windows.Forms.Timer
$timer.Interval=1000
$timer.Add_Tick({
    try {
        if ($ParentPid -gt 0 -and $null -eq (Get-Process -Id $ParentPid -ErrorAction SilentlyContinue)) {
            [System.Windows.Forms.Application]::ExitThread()
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
        $script:lastGame=$script:gameDisplay
        $gui=Get-Process -Id $GuiPid -ErrorAction SilentlyContinue
        if($null -eq $gui){$panel.Hide();return}
        $gui.Refresh()
        $hwnd=$gui.MainWindowHandle
        if($hwnd -eq [IntPtr]::Zero -or [EPStatusNative]::IsIconic($hwnd)){$panel.Hide();return}
        $fg=[EPStatusNative]::GetForegroundWindow()
        [uint32]$fgpid=0
        [void][EPStatusNative]::GetWindowThreadProcessId($fg,[ref]$fgpid)
        # Show the status tile only with the EssieuPilot application, not over other apps.
        if ($fgpid -ne $GuiPid -and $fgpid -ne [uint32]$PID) {$panel.Hide();return}
        $rect=New-Object EPStatusNative+RECT
        if(-not [EPStatusNative]::GetWindowRect($hwnd,[ref]$rect)){$panel.Hide();return}
        $panel.Left=[Math]::Max($rect.Left+10,$rect.Right-$panel.Width-26)
        $panel.Top=[Math]::Max($rect.Top+35,$rect.Bottom-$panel.Height-48)
        if(-not $panel.Visible){$panel.Show()}
    } catch {
        # Keep the main GUI usable even if the optional status panel fails.
        try{$panel.Hide()}catch{}
    }
})
try{
    $timer.Start()
    [System.Windows.Forms.Application]::Run($context)
} finally{
    $timer.Stop()
    $timer.Dispose()
    $tray.Visible=$false
    $tray.Dispose()
    $panel.Dispose()
    $menu.Dispose()
    $tooltip.Dispose()
    $context.Dispose()
}
