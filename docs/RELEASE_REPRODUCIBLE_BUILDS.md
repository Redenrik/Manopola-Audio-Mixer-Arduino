# Release Checksums and Reproducible Build Notes

This document defines the minimum repeatable release flow for MAMA artifacts.

## Goals

- Artifacts are built from a tagged commit with a clean working tree.
- Artifact names and folder layout are deterministic.
- A SHA-256 manifest is generated and can be independently verified.

## Baseline Requirements

- Use Go `1.24.x` for release builds.
- Build from a clean checkout (`git status --short` must be empty).
- Build from a tagged commit on `main`.

## Build Steps

### Automated cross-platform release packaging

Use `.github/workflows/release-artifacts.yml` for tagged releases. It builds portable archives for Linux/macOS/Windows and optionally emits Windows installer + update manifest assets.

Use `.github/workflows/release-notes.yml` to generate deterministic release-note markdown from git history and publish it back to the GitHub release.

For local Windows-only packaging from repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1
```

Optional installer from the portable directory:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-portable -AppVersion v1.0.0
```

Output folders:

- `dist\mama-portable\`
- `dist\installer\` (optional, requires Inno Setup `iscc`)

### Generate SHA-256 checksums

From repository root:

```bash
scripts/release/generate-checksums.sh dist
```

This writes:

- `dist/SHA256SUMS.txt`

### Generate release notes

```bash
scripts/release/generate-release-notes.sh v1.0.0
```

Optionally provide an explicit previous tag to pin the comparison range:

```bash
scripts/release/generate-release-notes.sh v1.0.0 v0.9.0
```

This writes `release-notes-v1.0.0.md` by default, grouped by conventional-commit prefixes when available.

### Sign release artifacts

From repository root:

```bash
scripts/release/sign-artifacts.sh dist/mama-portable
```

This produces Sigstore keyless signatures and certificates (`*.sig`, `*.pem`) for each file in the release directory.

## Verification Steps

From the artifact directory:

```bash
sha256sum -c SHA256SUMS.txt
```

On systems without `sha256sum`, use:

```bash
shasum -a 256 -c SHA256SUMS.txt
```

## Reproducibility Checklist

- Same source commit/tag used.
- Same Go minor line (`1.24.x`) used.
- Same packaging script (`scripts/windows/package-portable.ps1`) used.
- Generated checksums match previous build output.

If checksums differ, compare binary metadata and build environment before publishing.

For release workflow automation and macOS notarization details, see [docs/SIGNING_AND_NOTARIZATION.md](docs/SIGNING_AND_NOTARIZATION.md).


### Generate optional update manifests

```bash
scripts/release/generate-update-manifest.sh dist/mama-windows-amd64-portable.zip v1.0.0 https://github.com/<org>/<repo>/releases/download/v1.0.0/mama-windows-amd64-portable.zip
```

This writes `update-manifest.json` containing version, URL, checksum, and artifact size for updater integrations.
