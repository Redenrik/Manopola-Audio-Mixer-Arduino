#define MyAppName "MAMA"
#define MyAppPublisher "MAMA Contributors"
#ifndef MyAppVersion
  #define MyAppVersion "0.1.0"
#endif
#ifndef SourceDir
  #define SourceDir "dist\\mama-portable"
#endif
#ifndef OutputDir
  #define OutputDir "dist\\installer"
#endif

[Setup]
AppId={{2B2F9853-0E03-47B3-8A01-A4C593DE6437}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputDir={#OutputDir}
OutputBaseFilename=MAMA-Setup-{#MyAppVersion}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs

[Icons]
Name: "{group}\\MAMA Setup UI"; Filename: "{app}\\Open Setup UI.cmd"
Name: "{group}\\MAMA Runtime"; Filename: "{app}\\Start Mixer.cmd"
Name: "{commondesktop}\\MAMA Setup UI"; Filename: "{app}\\Open Setup UI.cmd"; Tasks: desktopicon

[Run]
Filename: "{app}\\Open Setup UI.cmd"; Description: "Launch setup UI"; Flags: shellexec nowait postinstall skipifsilent
