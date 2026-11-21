#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")
#if EP_PUBLISH_DIR == ""
  #error "EP_PUBLISH_DIR manquant"
#endif
#if EP_OUT_DIR == ""
  #error "EP_OUT_DIR manquant"
#endif
#if EP_APP_VERSION == ""
  #error "EP_APP_VERSION manquant"
#endif
[Setup]
AppId=ximisch.EssieuPilot.Beta.Reconstruction
AppName=EssieuPilot
AppVerName=EssieuPilot (Bêta {#EP_APP_VERSION})
AppVersion={#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot - Bêta (séparée)
DefaultDirName={localappdata}\Programs\EssieuPilot-Beta
DefaultGroupName=EssieuPilot (Bêta)
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot_Beta
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupLogging=yes
[Files]
Source: "{#EP_PUBLISH_DIR}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
[Icons]
Name: "{autoprograms}\EssieuPilot (Bêta)"; Filename: "{app}\EssieuPilot.exe"
Name: "{autodesktop}\EssieuPilot (Bêta)"; Filename: "{app}\EssieuPilot.exe"; Tasks: desktopicon
[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci bêta sur le Bureau"; GroupDescription: "Raccourcis :"; Flags: unchecked
[Run]
Filename: "{app}\EssieuPilot.exe"; Description: "Ouvrir EssieuPilot (Bêta)"; Flags: nowait postinstall skipifsilent
