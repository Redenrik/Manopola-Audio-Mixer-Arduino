# Manopola Audio Mixer for Arduino (MAMA)

MAMA is a physical USB audio mixer (Arduino master/slave) plus a single Go app that handles runtime mixing, setup UI, and live reconfiguration.

## What You Get

- Single app binary: `mama` (`mama.exe` on Windows)
- Live serial-driven knob control (`E<id>:+/-n`, `B<id>:1`)
- Target types: `master_out`, `mic_in`, `line_in`, `app`, `group`
- Smart groups and custom templates in the UI
- Immediate apply while editing mappings

## End-User Downloads

Recommended release assets:

- Windows 64-bit: `MAMA-Setup-Windows.exe`
- Windows 32-bit: `MAMA-Setup-Windows-32bit.exe`
- macOS universal2 installer: `MAMA-macOS.pkg`
- Linux amd64 installer: `MAMA-Linux.deb`

Advanced/power-user assets:

- Windows portable zip bundles (`amd64`, `386`)
- macOS portable (`amd64`, `arm64`)
- Linux arm64 portable
- Recommended portable bundles are also published: `MAMA-macOS.tar.gz`, `MAMA-Linux.tar.gz`

## Quick Start (No Coding)

### Windows

1. Install with `MAMA-Setup-Windows.exe` (or `MAMA-Setup-Windows-32bit.exe` on 32-bit systems).
2. Launch **MAMA Setup UI** from Start Menu.
3. In Settings, auto-detect/test the serial port.
4. Map knobs and click **Save Config**.
5. Keep MAMA running (close hides to tray, tray menu provides Show/Quit).

### macOS / Linux

1. Preferred: install with `MAMA-macOS.pkg` (macOS) or `MAMA-Linux.deb` (Linux).
2. Alternative: extract `MAMA-macOS.tar.gz` or `MAMA-Linux.tar.gz` to a writable folder.
3. Start setup UI launcher:
   - macOS: `MAMA.app` (double-click, no terminal)
   - Linux: `open-setup-ui.sh`
4. Configure serial + mappings, then save.
5. Optional runtime launcher:
   - macOS: `Start Mixer.command`
   - Linux: `start-mixer.sh`
6. Stop background runtime when needed:
   - macOS: `Stop Mixer.command`
   - Linux: `stop-mixer.sh`

Notes:
- Windows/macOS use embedded desktop-shell UI windows.
- Linux uses desktop-shell runtime mode with a local browser UI (`http://127.0.0.1:18765`).
- `Start Mixer` / `start-mixer.sh` launches MAMA in background and stores PID in `.mama.pid`.
- `Stop Mixer` / `stop-mixer.sh` uses `.mama.pid` to stop background runtime.
- Windows uses embedded desktop UI + tray by default.
- Linux uses native tray controls in desktop mode.
- On macOS, `mic_in`/`line_in` control the default system input level.
- On macOS, `app`/`group` session targets depend on an available per-app audio backend and may be unavailable.
- On macOS, programmatic master-volume updates do not trigger the keyboard volume HUD overlay.
- Startup-at-login can be enabled from Settings on Windows, macOS, and Linux.

## Hardware and Firmware

- Master board: `firmware/master/master.ino`
- Slave board: `firmware/slave/slave.ino`
- Protocol hello: `MAMA:HELLO:1`
- Legacy hello accepted by host: `V:1`
- Default baud: `115200`

## Config Basics

Default config discovery order:

1. `./config.yaml`
2. `mama/internal/config/default.yaml`

Most important mapping fields:

- `knob`
- `target`
- `step` (0..1, where `0.02` means 2% per detent)
- `selector` (`app`) / `selectors` (`group`)
- `fallback_target` + `fallback_name` (optional app/group fallback)

Full schema and examples: [docs/CONFIG_REFERENCE.md](docs/CONFIG_REFERENCE.md)

## Advanced CLI Usage

From `mama/`:

```bash
go run ./cmd/mama --help
```

Key flags:

- `-config` path to YAML config
- `-listen` HTTP bind (default `127.0.0.1:18765`)
- `-open` auto-open browser UI
- `-desktop` desktop shell mode (embedded shell on Windows/macOS, browser shell on Linux)
- `-start-hidden` start hidden shell (Windows: tray hidden; Linux: runtime starts without opening browser; macOS embedded mode starts minimized)

## Developer Quick Start

```bash
cd mama
go test ./...
go run ./cmd/mama
```

Or via make wrappers:

```bash
make test
make verify
make release-readiness
```

## Documentation Index

- [docs/README.md](docs/README.md)
- [docs/INSTALLATION.md](docs/INSTALLATION.md)
- [docs/CONFIG_REFERENCE.md](docs/CONFIG_REFERENCE.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [docs/PRODUCTION_READINESS.md](docs/PRODUCTION_READINESS.md)
- [docs/RELEASE_REPRODUCIBLE_BUILDS.md](docs/RELEASE_REPRODUCIBLE_BUILDS.md)
- [docs/SIGNING_AND_NOTARIZATION.md](docs/SIGNING_AND_NOTARIZATION.md)
- [docs/SUPPORT_POLICY.md](docs/SUPPORT_POLICY.md)
- [docs/SOAK_TEST_PLAN.md](docs/SOAK_TEST_PLAN.md)
- [SECURITY.md](SECURITY.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT. See [LICENSE](LICENSE).
