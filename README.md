# Manopola Audio Mixer for Arduino (MAMA)

MAMA is a physical USB audio mixer built with Arduino Nano boards and a Go desktop app.
The desktop app is a single executable (`mama`) that runs:
- the runtime mixer engine
- the setup UI
- tray/background mode

## What MAMA Supports

- 5+ knob event pipeline over serial (`E<id>:+/-n`, `B<id>:1`)
- target types: `master_out`, `mic_in`, `line_in`, `app`, `group`
- profile-based mappings with priorities/selectors
- live target discovery for devices/apps/groups
- immediate apply while editing mappings

## End-User Downloads

Recommended release assets:
- Windows 64-bit: `MAMA-Setup-Windows.exe`
- Windows 32-bit: `MAMA-Setup-Windows-32bit.exe`
- macOS (universal2): `MAMA-macOS.tar.gz`
- Linux (amd64): `MAMA-Linux.tar.gz`

Advanced assets are also published for power users:
- Linux arm64 portable
- macOS amd64/arm64 portable
- Windows amd64/386 portable zip bundles

## Quick Start (End Users)

### Windows

1. Download the correct installer:
   - 64-bit Windows: `MAMA-Setup-Windows.exe`
   - 32-bit Windows: `MAMA-Setup-Windows-32bit.exe`
2. Run the installer.
3. Launch **MAMA Setup UI**.
4. Set serial port, map knobs, and save.
5. Keep MAMA running in tray.

### macOS / Linux

1. Download and extract your OS package.
2. Run setup launcher:
   - macOS: `Open Setup UI.command`
   - Linux: `open-setup-ui.sh`
3. Save mappings.
4. Start hidden runtime launcher:
   - macOS: `Start Mixer.command`
   - Linux: `start-mixer.sh`

## Developer Quick Start

```bash
cd mama
go test ./...
go run ./cmd/mama
```

If you use `make`:

```bash
make test
make verify
make release-readiness
```

## Serial Protocol

Firmware emits newline-terminated lines:
- `MAMA:HELLO:1`
- `E1:+1`
- `E1:-1`
- `B1:1`

Host compatibility:
- supports protocol version `1`
- accepts legacy `V:1` for backward compatibility
- rejects unsupported announced versions safely

## Configuration

Default config path order:
1. `./config.yaml`
2. `mama/internal/config/default.yaml`

Primary mapping fields:
- `knob`
- `target`
- `step`
- `selector` / `selectors` for `app` and `group`
- optional `priority` for overlap resolution

For full install/build/release details, see docs below.

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/INSTALLATION.md](docs/INSTALLATION.md)
- [docs/PRODUCTION_READINESS.md](docs/PRODUCTION_READINESS.md)
- [docs/RELEASE_REPRODUCIBLE_BUILDS.md](docs/RELEASE_REPRODUCIBLE_BUILDS.md)
- [docs/SIGNING_AND_NOTARIZATION.md](docs/SIGNING_AND_NOTARIZATION.md)
- [docs/SOAK_TEST_PLAN.md](docs/SOAK_TEST_PLAN.md)
- [docs/SUPPORT_POLICY.md](docs/SUPPORT_POLICY.md)
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [SECURITY.md](SECURITY.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT. See `LICENSE`.
