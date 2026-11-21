#define EP_PUBLISH_DIR GetEnv("EP_PUBLISH_DIR")
#define EP_OUT_DIR GetEnv("EP_OUT_DIR")
#define EP_APP_VERSION GetEnv("EP_APP_VERSION")
; This path is set by the release pipeline ONLY AFTER physical DLL validation.
; Empty means an app-only installer: no untested DLL may reach ATS.
#define EP_DLL_PAYLOAD GetEnv("EP_VALIDATED_DLL_PAYLOAD")
#if EP_DLL_PAYLOAD != ""
  #if !FileExists(AddBackslash(EP_DLL_PAYLOAD) + "module-manifest.json")
    #error "Validated DLL payload is missing its manifest"
  #endif
  #if !FileExists(AddBackslash(EP_DLL_PAYLOAD) + "EssieuPilotATS.dll")
    #error "Validated physical DLL missing"
  #endif
  #if !FileExists(AddBackslash(EP_DLL_PAYLOAD) + "scs_telemetry_hub.dll")
    #error "Validated telemetry DLL missing"
  #endif
#endif

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
AppId=ximisch.EssieuPilot.Original.UI.Test
AppName=EssieuPilot
AppVerName=EssieuPilot
AppVersion={#EP_APP_VERSION}
AppPublisher=EssieuPilot
UninstallDisplayName=EssieuPilot - essai interface originale
DefaultDirName={localappdata}\Programs\EssieuPilot-OriginalTest
DefaultGroupName=EssieuPilot - essai interface originale
DisableProgramGroupPage=yes
OutputDir={#EP_OUT_DIR}
OutputBaseFilename=Installer_EssieuPilot
Compression=lzma2
SolidCompression=yes
#if EP_DLL_PAYLOAD != ""
PrivilegesRequired=admin
#else
PrivilegesRequired=lowest
#endif
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupLogging=yes
UninstallDisplayIcon={app}\EssieuPilot.exe
[Languages]
Name: "french"; MessagesFile: "compiler:Languages\French.isl"
[Files]
Source: "{#EP_PUBLISH_DIR}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
#if EP_DLL_PAYLOAD != ""
; A vetted release carries its precise DLLs and a digest manifest.
; Module management belongs to the SAME installer invoked by in-app updates.
Source: "{#AddBackslash(SourcePath)}..\integration\installer\Manage-ATS-Plugins-V2.ps1"; DestDir: "{app}\ATS-Plugin"; Flags: ignoreversion
Source: "{#EP_DLL_PAYLOAD}\module-manifest.json"; DestDir: "{app}\ATS-Plugin"; Flags: ignoreversion
Source: "{#EP_DLL_PAYLOAD}\EssieuPilotATS.dll"; DestDir: "{app}\ATS-Plugin"; Flags: ignoreversion
Source: "{#EP_DLL_PAYLOAD}\scs_telemetry_hub.dll"; DestDir: "{app}\ATS-Plugin"; Flags: ignoreversion
; Optional manifest-listed EssieuPilot .scs addons go to the user ATS mods folder
; via the same transaction, NEVER to game data or core *.scs files.
Source: "{#EP_DLL_PAYLOAD}\EssieuPilot_*.scs"; DestDir: "{app}\ATS-Plugin"; Flags: ignoreversion skipifsourcedoesntexist
#endif
[Tasks]
Name: "desktopicon"; Description: "Créer un raccourci sur le Bureau"; GroupDescription: "Raccourcis facultatifs"; Flags: unchecked
#if EP_DLL_PAYLOAD != ""
Name: "atsplugin"; Description: "Installer / mettre à jour automatiquement les DLL et mods .scs validés pour ATS"; GroupDescription: "Intégration American Truck Simulator :"
#endif
[Icons]
Name: "{group}\EssieuPilot"; Filename: "{app}\EssieuPilot.exe"
Name: "{autodesktop}\EssieuPilot"; Filename: "{app}\EssieuPilot.exe"; Tasks: desktopicon
#if EP_DLL_PAYLOAD != ""
Name: "{group}\EssieuPilot - vérifier les DLL ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins-V2.ps1"" -Action Status"
Name: "{group}\EssieuPilot - retirer les DLL ATS"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\ATS-Plugin\Manage-ATS-Plugins-V2.ps1"" -Action Remove"
#endif
[Run]
Filename: "{app}\EssieuPilot.exe"; Description: "Ouvrir EssieuPilot"; Flags: nowait postinstall skipifsilent
; Test channel isolated from official v1.6.2 and other experiments.
; Profiles in AppData are never removed when upgrading/uninstalling.
; App-only builds never modify ATS. For validated, explicitly packaged DLLs,
; this installer performs first-install/update/rollback for DLL and SCS automatically.
; Other plugins and original UI profiles are never changed.
#if EP_DLL_PAYLOAD != ""
[Code]
function PrepareToInstall(var NeedsRestart: Boolean): String;
var ResultCode: Integer;
begin
  Result := '';
  if IsTaskSelected('atsplugin') then begin
    if (not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
      '-NoProfile -NonInteractive -Command "if(Get-Process -Name amtrucks -ErrorAction SilentlyContinue){exit 8}else{exit 0}"',
      '', SW_HIDE, ewWaitUntilTerminated, ResultCode)) or (ResultCode <> 0) then
      Result := 'Ferme American Truck Simulator avant de mettre à jour les DLL. L’installation est interrompue pour protéger le jeu.';
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var ResultCode: Integer;
begin
  if (CurStep = ssPostInstall) and IsTaskSelected('atsplugin') then begin
    if (not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
      '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' +
        ExpandConstant('{app}\ATS-Plugin\Manage-ATS-Plugins-V2.ps1') +
      '" -Action Install',
      '', SW_SHOW, ewWaitUntilTerminated, ResultCode)) or (ResultCode <> 0) then
      MsgBox('EssieuPilot a été installé, mais la synchronisation des DLL ATS a échoué.' +
        Chr(13) + Chr(10) + 'Les autres plugins ATS sont conservés. Ferme ATS et vérifie son dossier Steam.' +
        Chr(13) + Chr(10) + 'Aucune DLL non reconnue n’a été écrasée.', mbError, MB_OK);
  end;
end;
#endif
