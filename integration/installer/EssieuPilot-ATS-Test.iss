#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")
#if EP_PUBLISH_DIR == ""
  #error "EP_PUBLISH_DIR must point to the packaged Windows app"
#endif
#if EP_OUT_DIR == ""
  #error "EP_OUT_DIR required"
#endif
#if EP_APP_VERSION == ""
  #error "EP_APP_VERSION required"
#endif

[Setup]
AppId=ximisch.EssieuPilot.Beta.Reconstruction
AppName=EssieuPilot
AppVersion={#EP_APP_VERSION}
AppVerName=EssieuPilot — Essai ATS {#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot — Essai ATS (séparé de 1.6.2)
DefaultDirName={localappdata}\Programs\EssieuPilot-Beta
DefaultGroupName=EssieuPilot
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot_ATS_Test
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=admin
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupLogging=yes
SetupIconFile={#EP_PUBLISH_DIR}\EssieuPilot.ico
UninstallDisplayIcon={app}\EssieuPilot.exe
DisableWelcomePage=no
[Languages]
Name: "french"; MessagesFile: "compiler:Languages\French.isl"

[Files]
Source: "{#EP_PUBLISH_DIR}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci EssieuPilot sur le Bureau"; GroupDescription: "Options :"; Flags: unchecked
Name: "atsplugin"; Description: "Installer les DLL dans ATS 1.62 (EXPERIMENTAL : risque de crash, solo DirectX 11 uniquement)"; GroupDescription: "Module ATS :"; Flags: unchecked

[Icons]
Name: "{autodesktop}\EssieuPilot (Essai ATS)"; Filename: "{app}\EssieuPilot.exe"; Tasks: desktopicon
Name: "{autoprograms}\EssieuPilot"; Filename: "{app}\EssieuPilot.exe"
Name: "{autoprograms}\EssieuPilot - installer les DLL ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins.ps1"" -Action Install"
Name: "{autoprograms}\EssieuPilot - retirer les DLL ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins.ps1"" -Action Remove"

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins.ps1"" -Action Install"; WorkingDir: "{app}\ATS-Plugin"; Description: "Installer les DLL EssieuPilot dans ATS"; Tasks: atsplugin; Flags: waituntilterminated
Filename: "{app}\EssieuPilot.exe"; Description: "Ouvrir EssieuPilot"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins.ps1"" -Action Remove"; Flags: runhidden
