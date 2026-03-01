# Installation and Packaging

## Audience Split

- End users: should not need terminal usage.
- Maintainers: generate release artifacts from source.

## End User Install (Planned Release UX)

Recommended distribution format: portable ZIP.

User steps:
1. Download release ZIP.
2. Extract to any folder (for example `Desktop\MAMA`).
3. Double-click `Open Setup UI.cmd`.
4. Select serial port, assign mappings, save.
5. Double-click `Start Mixer.cmd`.

No service installation required.

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

## Non-Pollution Principles

Default package behavior:
- no write outside package directory except normal OS temp/log internals
- no registry requirement for normal operation
- no scheduled task/service auto-install
- uninstall is deleting the folder

## Optional Installer (Future)

If you later add an installer:
- keep a portable mode option
- explicit opt-in for auto-start
- clear uninstall path
- avoid forced privileged install unless necessary

## Linux/macOS Notes

Current scripts target Windows first.

For Linux/macOS maintainers:
- build `mama` and `mama-ui` with `go build`
- place both binaries and `config.yaml` in one directory
- create desktop launchers (`.desktop`/app bundle) if needed
