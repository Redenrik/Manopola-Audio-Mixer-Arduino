# Installation and Packaging

## End Users (No Terminal)

Download from the latest release:

- Windows 64-bit: `MAMA-Setup-Windows.exe`
- Windows 32-bit: `MAMA-Setup-Windows-32bit.exe`
- macOS: `MAMA-macOS.tar.gz`
- Linux: `MAMA-Linux.tar.gz`

### Windows

1. Choose the installer matching your OS architecture (64-bit or 32-bit).
2. Run installer.
3. Open **MAMA Setup UI**.
4. Set serial port and knob mappings.
5. Save and keep MAMA running in tray.

### macOS / Linux

1. Extract package to a writable folder.
2. Run setup launcher:
   - macOS: `Open Setup UI.command`
   - Linux: `open-setup-ui.sh`
3. Save mappings.
4. Run mixer launcher:
   - macOS: `Start Mixer.command`
   - Linux: `start-mixer.sh`

## Maintainers

### Windows installers (64-bit + 32-bit)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-amd64-portable -Arch amd64 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-amd64-portable -OutputDir dist\installer-amd64 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup

powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-386-portable -Arch 386 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-386-portable -OutputDir dist\installer-386 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup-Windows-32bit
```

### Linux portable builds

```bash
bash scripts/release/package-portable.sh linux amd64 dist/MAMA-Linux
bash scripts/release/package-portable.sh linux arm64 dist/mama-linux-arm64-portable
```

### macOS portable builds

```bash
bash scripts/release/package-macos-universal.sh dist/MAMA-macOS
bash scripts/release/package-portable.sh darwin amd64 dist/mama-macos-amd64-portable
bash scripts/release/package-portable.sh darwin arm64 dist/mama-macos-arm64-portable
```

### Optional advanced Windows portable zips

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

For reproducible release details, see [RELEASE_REPRODUCIBLE_BUILDS.md](RELEASE_REPRODUCIBLE_BUILDS.md).
