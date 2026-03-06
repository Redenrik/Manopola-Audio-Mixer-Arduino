# Installation and Packaging

## End Users (No Terminal)

Download from the latest release:

- Windows 64-bit: `MAMA-Setup-Windows.exe`
- Windows 32-bit: `MAMA-Setup-Windows-32bit.exe`
- macOS universal2 installer: `MAMA-macOS.pkg`
- Linux amd64 installer: `MAMA-Linux.deb`
- macOS portable distributable: `MAMA-macOS.tar.gz`
- Linux portable distributable: `MAMA-Linux.tar.gz`

### Windows (Recommended Installer)

1. Run the installer matching your OS architecture.
2. Launch **MAMA Setup UI** from Start Menu.
3. In Settings:
   - click **Auto-Detect MAMA**
   - click **Test Connection**
   - configure mappings
4. Click **Save Config**.
5. Keep MAMA running in tray (close hides window; tray menu has Show/Quit).

### macOS (Recommended Installer)

1. Open `MAMA-macOS.pkg` and complete installation.
2. Launch setup from `/Applications/MAMA/Open Setup UI.command`.
3. Configure serial + mappings and save.

### Linux (Recommended Installer)

1. Install package: `sudo apt install ./MAMA-Linux.deb`.
2. Run setup:
   - app menu entry: **MAMA Audio Mixer**
   - or terminal: `mama-setup`
3. Configure serial + mappings and save.

### macOS / Linux (Portable Package)

1. Extract package to a writable folder.
2. Start setup:
   - macOS: `Open Setup UI.command`
   - Linux: `open-setup-ui.sh`
3. Configure serial + mappings and save.
4. Optional runtime launcher:
   - macOS: `Start Mixer.command`
   - Linux: `start-mixer.sh`
5. Optional stop launcher:
   - macOS: `Stop Mixer.command`
   - Linux: `stop-mixer.sh`

Notes:
- On macOS/Linux the app runs in desktop-shell browser mode (`127.0.0.1` only).
- In desktop mode, macOS/Linux provide tray/menu-bar controls (desktop session required).
- `Start Mixer` / `start-mixer.sh` disables auto-open browser and runs MAMA in background.
- background PID is written to `.mama.pid` in the package folder.
- `Stop Mixer` / `stop-mixer.sh` stops the PID tracked in `.mama.pid`.
- Startup-at-login can be enabled from the app Settings page (Windows/macOS/Linux).

## Maintainers: Build Release Artifacts

### Windows installers (`amd64` + `386`)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-amd64-portable -Arch amd64 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-amd64-portable -OutputDir dist\installer-amd64 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup

powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-386-portable -Arch 386 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-386-portable -OutputDir dist\installer-386 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup-Windows-32bit
```

### Linux packages

```bash
bash scripts/release/package-portable.sh linux amd64 dist/MAMA-Linux
tar -C dist -czf dist/MAMA-Linux.tar.gz MAMA-Linux
bash scripts/release/package-linux-deb.sh dist/MAMA-Linux v1.0.0 dist/MAMA-Linux.deb
```

Optional advanced Linux arm64:

```bash
bash scripts/release/package-portable.sh linux arm64 dist/mama-linux-arm64-portable
tar -C dist -czf dist/mama-linux-arm64-portable.tar.gz mama-linux-arm64-portable
```

### macOS packages

```bash
bash scripts/release/package-macos-universal.sh dist/MAMA-macOS
tar -C dist -czf dist/MAMA-macOS.tar.gz MAMA-macOS
bash scripts/release/package-macos-pkg.sh dist/MAMA-macOS v1.0.0 dist/MAMA-macOS.pkg
```

Optional advanced macOS portable builds:

```bash
bash scripts/release/package-portable.sh darwin amd64 dist/mama-macos-amd64-portable
bash scripts/release/package-portable.sh darwin arm64 dist/mama-macos-arm64-portable
```

### Optional advanced Windows portable zip bundles

```powershell
Compress-Archive -Path dist\mama-windows-amd64-portable\* -DestinationPath dist\mama-windows-amd64-portable.zip -Force
Compress-Archive -Path dist\mama-windows-386-portable\* -DestinationPath dist\mama-windows-386-portable.zip -Force
```

## Integrity

Generate checksums:

```bash
bash scripts/release/generate-checksums.sh dist
```

Generate update manifest:

```bash
bash scripts/release/generate-update-manifest.sh <artifact-path> <version> <download-url>
```

See [RELEASE_REPRODUCIBLE_BUILDS.md](RELEASE_REPRODUCIBLE_BUILDS.md) for full release workflow.
