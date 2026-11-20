# EssieuPilot 1.6.3 RC2 - non-invasive ATS process status in Windows notification area.
# Does not load a DLL, touch game files or change EssieuPilot user profiles.
param(
    [int]$ParentPid = 0,
    [switch]$SelfTest
)
$ErrorActionPreference = 'Stop'

function Get-AtsRunning {
    param([scriptblock]$Probe = { @(Get-Process -Name 'amtrucks' -ErrorAction SilentlyContinue).Count -gt 0 })
    try {
        $value = & $Probe
        if ($value -isnot [bool]) { return $null }
        return $value
    } catch {
        return $null
    }
}

if ($SelfTest) {
    if ((Get-AtsRunning -Probe { $true }) -ne $true) { throw 'Running detection failed' }
    if ((Get-AtsRunning -Probe { $false }) -ne $false) { throw 'Stopped detection failed' }
    if ($null -ne (Get-AtsRunning -Probe { throw 'Access error' })) { throw 'Unknown state must not be reported as stopped' }
    Write-Output 'PASS: ATS process present, absent, and unknown without a DLL'
    exit 0
}

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()
$context = New-Object System.Windows.Forms.ApplicationContext
$tray = New-Object System.Windows.Forms.NotifyIcon
$tray.Icon = [System.Drawing.SystemIcons]::Application
$tray.Text = 'EssieuPilot - verification ATS'
$tray.Visible = $true

$menu = New-Object System.Windows.Forms.ContextMenuStrip
$status = New-Object System.Windows.Forms.ToolStripMenuItem('ATS : verification...')
$status.Enabled = $false
[void]$menu.Items.Add($status)
[void]$menu.Items.Add((New-Object System.Windows.Forms.ToolStripSeparator))
$details = New-Object System.Windows.Forms.ToolStripMenuItem('Afficher le statut ATS')
$details.Add_Click({
    [void][System.Windows.Forms.MessageBox]::Show(
        ($script:displayStatus + "`r`nDetection de amtrucks.exe (sans DLL ni telemetrie)."),
        'EssieuPilot - Detection ATS',
        [System.Windows.Forms.MessageBoxButtons]::OK,
        [System.Windows.Forms.MessageBoxIcon]::Information)
})
[void]$menu.Items.Add($details)
$close = New-Object System.Windows.Forms.ToolStripMenuItem('Fermer le diagnostic ATS')
$close.Add_Click({ [System.Windows.Forms.Application]::ExitThread() })
[void]$menu.Items.Add($close)
$tray.ContextMenuStrip = $menu
$tray.Add_DoubleClick({
    [void][System.Windows.Forms.MessageBox]::Show(
        $script:displayStatus, 'EssieuPilot - Detection ATS')
})

$script:displayStatus = 'Etat indisponible'
$script:lastState = 'init'
$timer = New-Object System.Windows.Forms.Timer
$timer.Interval = 1000
$update = {
    if ($ParentPid -gt 0) {
        if ($null -eq (Get-Process -Id $ParentPid -ErrorAction SilentlyContinue)) {
            [System.Windows.Forms.Application]::ExitThread()
            return
        }
    }
    $state = Get-AtsRunning
    $stateKey = if ($null -eq $state) { 'unknown' } elseif ($state) { 'running' } else { 'stopped' }
    $script:displayStatus = switch ($stateKey) {
        'running' { 'ATS LANCE' }
        'stopped' { 'ATS NON LANCE' }
        default { 'ETAT ATS INDISPONIBLE' }
    }
    $status.Text = $script:displayStatus
    $tray.Text = ('EssieuPilot - ' + $script:displayStatus)
    if ($script:lastState -ne 'init' -and $script:lastState -ne $stateKey) {
        try {
            $tray.ShowBalloonTip(2500, 'EssieuPilot', $script:displayStatus,
                [System.Windows.Forms.ToolTipIcon]::Info)
        } catch { }
    }
    $script:lastState = $stateKey
}
$timer.Add_Tick($update)
try {
    & $update
    $timer.Start()
    [System.Windows.Forms.Application]::Run($context)
} finally {
    $timer.Stop()
    $timer.Dispose()
    $tray.Visible = $false
    $tray.Dispose()
    $menu.Dispose()
    $context.Dispose()
}
