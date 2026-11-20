#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")
#if EP_PUBLISH_DIR == ""
  #error "EP_PUBLISH_DIR required"
#endif
#if EP_OUT_DIR == ""
  #error "EP_OUT_DIR required"
#endif
#if EP_APP_VERSION == ""
  #error "EP_APP_VERSION required"
#endif

[Setup]
AppId=ximisch.EssieuPilot.Native.Windows
AppName=EssieuPilot
AppVerName=EssieuPilot
AppVersion={#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot
DefaultDirName={localappdata}\Programs\EssieuPilot-Native
DefaultGroupName=EssieuPilot - essai natif
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupLogging=yes

[Languages]
Name: "french"; MessagesFile: "compiler:Languages\French.isl"

[Files]
Source: "{#EP_PUBLISH_DIR}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci sur le Bureau"; GroupDescription: "Raccourcis"; Flags: unchecked

[Icons]
Name: "{group}\EssieuPilot"; Filename: "{app}\EssieuPilot.exe"
Name: "{autodesktop}\EssieuPilot"; Filename: "{app}\EssieuPilot.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\EssieuPilot.exe"; Description: "Ouvrir EssieuPilot"; Flags: nowait postinstall skipifsilent

; This native edition is in a different folder and AppId from all existing
; EssieuPilot versions. It never deletes %APPDATA%\EssieuPilot or profiles.
; It never copies any DLL to American Truck Simulator.
