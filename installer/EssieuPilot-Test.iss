#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")

#if EP_PUBLISH_DIR == ""
  #error "EP_PUBLISH_DIR absent."
#endif
#if EP_OUT_DIR == ""
  #error "EP_OUT_DIR absent."
#endif
#if EP_APP_VERSION == ""
  #error "EP_APP_VERSION absent."
#endif

[Setup]
AppId=ximisch.EssieuPilot.UpdateTest
AppName=EssieuPilot
AppVersion={#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot - canal de test
DefaultDirName={localappdata}\Programs\EssieuPilot-UpdateTest
DefaultGroupName=EssieuPilot - Test
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
SetupLogging=yes

[Files]
Source: "{#EP_PUBLISH_DIR}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\EssieuPilot - Test"; Filename: "{app}\EssieuPilot.exe"
Name: "{autodesktop}\EssieuPilot - Test"; Filename: "{app}\EssieuPilot.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci de test sur le bureau"; GroupDescription: "Raccourcis :"; Flags: unchecked

[Run]
Filename: "{app}\EssieuPilot.exe"; Description: "Ouvrir EssieuPilot"; Flags: nowait postinstall skipifsilent
