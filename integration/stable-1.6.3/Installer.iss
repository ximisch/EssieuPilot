#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")
#if EP_PUBLISH_DIR == ""
  #error "EP_PUBLISH_DIR absent"
#endif
#if EP_OUT_DIR == ""
  #error "EP_OUT_DIR absent"
#endif
#if EP_APP_VERSION == ""
  #error "EP_APP_VERSION absent"
#endif
[Setup]
AppId=ximisch.EssieuPilot.ATS.1.6.3.RC
AppName=EssieuPilot
AppVerName=EssieuPilot
AppVersion={#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot - Test ATS
DefaultDirName={localappdata}\Programs\EssieuPilot-ATS-Test
DefaultGroupName=EssieuPilot - Test ATS
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot_1.6.3_Essai
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupLogging=yes
SetupIconFile={#EP_PUBLISH_DIR}\EssieuPilot.ico
UninstallDisplayIcon={app}\EssieuPilot.exe

[Languages]
Name: "french"; MessagesFile: "compiler:Languages\French.isl"

[Files]
Source: "{#EP_PUBLISH_DIR}\EssieuPilot.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#EP_PUBLISH_DIR}\EssieuPilotController.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#EP_PUBLISH_DIR}\EssieuPilot.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#EP_PUBLISH_DIR}\Start-EssieuPilot.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#EP_PUBLISH_DIR}\LISEZ-MOI.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#EP_PUBLISH_DIR}\Plugins ATS\*"; DestDir: "{app}\Plugins ATS"; Flags: ignoreversion recursesubdirs createallsubdirs

[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci d'essai sur le Bureau"; GroupDescription: "Raccourci facultatif :"; Flags: unchecked

[Icons]
Name: "{autoprograms}\EssieuPilot - essai ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\Start-EssieuPilot.ps1"""; IconFilename: "{app}\EssieuPilot.exe"; WorkingDir: "{app}"
Name: "{autoprograms}\EssieuPilot - Plugins ATS"; Filename: "{app}\Plugins ATS"
Name: "{autoprograms}\EssieuPilot - Instructions"; Filename: "{app}\LISEZ-MOI.txt"
Name: "{autodesktop}\EssieuPilot - essai ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\Start-EssieuPilot.ps1"""; IconFilename: "{app}\EssieuPilot.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\Start-EssieuPilot.ps1"""; Description: "Ouvrir EssieuPilot"; Flags: nowait postinstall skipifsilent

; IMPORTANT: NO DLLs are copied into the ATS directory by this installer.
; Uninstall does not remove user profiles or other installed versions.
