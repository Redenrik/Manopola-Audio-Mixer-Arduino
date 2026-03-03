# Installation and Packaging

## Audience Split

- End users: should not need terminal usage.
- Maintainers: generate release artifacts from source.

## End User Install (No Terminal)

Recommended distribution format: portable ZIP.

User steps:
1. Download release ZIP from the project Releases page.
2. Extract to any folder (for example `Desktop\MAMA`).
3. Double-click `Open Setup UI.cmd`.
4. Select serial port, assign mappings, save.
5. (Optional) Enable **Launch mixer at OS startup** in Setup UI and click **Apply Startup Setting**.
6. Double-click `Start Mixer.cmd`.

No service installation required and no terminal commands are needed for this flow. Startup integration is available for Windows/macOS packaged launches.

## Maintainer Build (Windows)

From repo root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-portable.ps1
```

Output folder:
- `dist\mama-portable\`

Contents:
- `mama.exe`
- `mama-ui.exe`
- `config.yaml`
- `Open Setup UI.cmd`
- `Start Mixer.cmd`

## Release Integrity (Maintainers)

After generating `dist\mama-portable`, create a checksum manifest:

```bash
scripts/release/generate-checksums.sh dist/mama-portable
```

Publish `SHA256SUMS.txt` with release artifacts and verify it before publishing:

```bash
cd dist/mama-portable
sha256sum -c SHA256SUMS.txt
```

For the full reproducible-build checklist, see [`RELEASE_REPRODUCIBLE_BUILDS.md`](RELEASE_REPRODUCIBLE_BUILDS.md).

## Non-Pollution Principles

Default package behavior:
- no write outside package directory except normal OS temp/log internals
- no registry requirement for normal operation
- no scheduled task/service auto-install
- uninstall is deleting the folder

## Optional Installer (Windows Maintainers)

Portable ZIP remains the default distribution.

For maintainers that want an installer artifact in addition to portable mode:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows\package-installer.ps1 -PortableDir dist\mama-portable -AppVersion v1.0.0
```

Notes:
- installer generation is optional and requires Inno Setup (`iscc`)
- when `iscc` is unavailable, the script exits successfully after printing a skip warning
- the portable package remains the canonical fallback artifact

## Optional Update Manifest

To ship a machine-readable update payload descriptor for any archive:

```bash
scripts/release/generate-update-manifest.sh dist/mama-portable.zip v1.0.0 https://github.com/<org>/<repo>/releases/download/v1.0.0/mama-portable.zip
```

This writes `update-manifest.json` with version, URL, checksum, and size metadata.

## Linux/macOS Notes

Current scripts target Windows first.

For Linux/macOS maintainers:
- build `mama` and `mama-ui` with `go build`
- place both binaries and `config.yaml` in one directory
- create desktop launchers (`.desktop`/app bundle) if needed
