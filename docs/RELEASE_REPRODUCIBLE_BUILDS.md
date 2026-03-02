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

### Windows portable artifacts

From repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1
```

Output folder:

- `dist\mama-portable\`

### Generate SHA-256 checksums

From repository root:

```bash
scripts/release/generate-checksums.sh dist/mama-portable
```

This writes:

- `dist/mama-portable/SHA256SUMS.txt`

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
