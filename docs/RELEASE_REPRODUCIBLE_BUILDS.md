# Release Checksums and Reproducible Build Notes

## Goals

- Build artifacts from a tagged commit and clean working tree.
- Keep deterministic artifact naming.
- Publish SHA-256 checksums for independent verification.

## Baseline Requirements

- Go `1.22+` (CI currently runs `1.24.x`)
- clean checkout (`git status --short` empty)
- tagged commit on `main`

## Automated Release Workflow

Primary workflow:
- `.github/workflows/release-artifacts.yml`

Recommended user assets:
- `MAMA-Setup-Windows.exe` (Windows 64-bit)
- `MAMA-Setup-Windows-32bit.exe` (Windows 32-bit)
- `MAMA-macOS.tar.gz` (universal2)
- `MAMA-Linux.tar.gz` (amd64)

Advanced assets:
- Linux arm64 portable
- macOS amd64/arm64 portable
- Windows amd64/386 portable zips

## Local Maintainer Packaging

### Windows installers

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-amd64-portable -Arch amd64 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-amd64-portable -OutputDir dist\installer-amd64 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup

powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1 -OutputDir dist\mama-windows-386-portable -Arch 386 -BinaryName mama.exe
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-windows-386-portable -OutputDir dist\installer-386 -AppVersion v1.0.0 -OutputBaseName MAMA-Setup-Windows-32bit
```

### Linux/macOS packages

```bash
bash scripts/release/package-portable.sh linux amd64 dist/MAMA-Linux
bash scripts/release/package-macos-universal.sh dist/MAMA-macOS
```

## Checksums

Generate:

```bash
bash scripts/release/generate-checksums.sh dist
```

Verify:

```bash
sha256sum -c dist/SHA256SUMS.txt
```

If `sha256sum` is unavailable:

```bash
shasum -a 256 -c dist/SHA256SUMS.txt
```

## Update Manifests

```bash
bash scripts/release/generate-update-manifest.sh dist/MAMA-Setup-Windows.exe v1.0.0 https://github.com/<org>/<repo>/releases/download/v1.0.0/MAMA-Setup-Windows.exe
```

Manifest payload includes:
- version
- published timestamp
- URL
- SHA-256
- size

## Reproducibility Checklist

- same source commit/tag
- compatible Go line (`1.22+`; CI currently `1.24.x`)
- same packaging scripts
- matching checksum manifests
