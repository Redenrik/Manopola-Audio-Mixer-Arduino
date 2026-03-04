# Installation and Packaging

## Audience Split

- End users: install and run with minimal decisions.
- Maintainers: build deterministic release artifacts.

## End User Install (No Terminal)

Use the **recommended asset per OS** from the release page:

- Windows: `MAMA-Setup-Windows.exe`
- macOS: `MAMA-macOS.tar.gz`
- Linux: `MAMA-Linux.tar.gz`

Advanced architecture-specific assets remain available for power users (Linux/macOS `amd64` + `arm64`, Windows portable `amd64`).

### Windows (recommended)

1. Download `MAMA-Setup-Windows.exe`.
2. Run installer.
3. Launch **MAMA Setup UI** from Start menu/desktop shortcut.
4. Configure serial + knob mappings and save.
5. (Optional) Enable startup from Setup UI.

### macOS / Linux (recommended)

1. Download the OS package (`MAMA-macOS.tar.gz` or `MAMA-Linux.tar.gz`).
2. Extract to a writable folder.
3. Run setup launcher:
   - macOS: `Open Setup UI.command`
   - Linux: `open-setup-ui.sh`
4. Save config and then use runtime launcher:
   - macOS: `Start Mixer.command`
   - Linux: `start-mixer.sh`

## Maintainer Build Commands

### Windows recommended installer

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-amd64-portable -Arch amd64 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-amd64-portable -AppVersion v1.0.0
```

### Windows portable build

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-amd64-portable -Arch amd64 -BinaryName mama.exe
```

Note: native Windows `arm64` builds are currently blocked by upstream dependency constraints; Windows ARM devices should use the x64 installer/package.

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

## Release Integrity

Generate checksum manifests:

```bash
bash scripts/release/generate-checksums.sh dist
```

Generate update manifests:

```bash
bash scripts/release/generate-update-manifest.sh <artifact-path> <version> <download-url>
```

For the full reproducible-build flow, see [RELEASE_REPRODUCIBLE_BUILDS.md](RELEASE_REPRODUCIBLE_BUILDS.md).
