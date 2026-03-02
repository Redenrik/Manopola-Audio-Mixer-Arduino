# Manopola Audio Mixer for Arduino (MAMA)

MAMA is a physical USB audio mixer built with 2 Arduino Nano boards (master + slave) and a Go desktop app.

This repository now includes:
- firmware for 5 rotary encoders with push buttons
- runtime daemon (`mama`) that applies volume actions
- browser-based setup GUI (`mama-ui`) with visual knob identification and first-run guided wizard
- collaboration and release documentation for contributors and maintainers

## Goals

- Easy collaboration: clear docs, explicit architecture, predictable release flow.
- Easy for non-technical users: no manual YAML editing required for setup.
- Low system impact: portable-first operation, no required global services.
- Clear UX: a visual identify mode to map physical knobs to software mappings.

## Current Feature Status

- Implemented end-to-end:
  - 5 knob event pipeline (encoder deltas + button press events)
  - serial protocol parsing and config validation
  - `master_out` adjust + mute toggle on supported platforms
  - `mic_in` and `line_in` adjust + mute toggle (Windows + Unix hosts with capture endpoint tooling available)
  - setup UI (`mama-ui`) to edit serial settings/mappings and save config
  - setup UI serial connection test button for validating port + baud before saving
  - setup UI first-run guided wizard (detect board -> test port -> map knobs -> save -> verify)
  - setup UI live identify mode (rotate/press a knob and the matching indicator flashes)
  - setup UI mapping templates for streaming/conferencing/music/gaming presets
  - setup UI config backup/restore snapshot controls plus JSON import/export workflow
  - setup UI accessibility pass for keyboard focus visibility, screen-reader labels, and live status announcements
  - setup UI localization framework with English/Italian language toggle and persisted language preference
  - setup API `/api/targets` now returns discovered backend targets (`discovered`) alongside compatibility fields (`known`, `supported`)
- Additional target support:
  - `app` per-session volume + mute on Unix hosts with `pactl` sink-input controls
  - `group` grouped app/session volume + mute on Unix hosts with `pactl` sink-input controls

## Repository Layout

- `firmware/master/master.ino`: USB-connected master Arduino firmware (encoders 1-3 + I2C poll for 4-5).
- `firmware/slave/slave.ino`: I2C slave Arduino firmware (encoders 4-5).
- `mama/cmd/mama`: runtime daemon (serial events -> audio actions).
- `mama/cmd/mama-ui`: local setup GUI server.
- `mama/internal/*`: config, protocol parser, serial I/O, audio backend, setup UI server.
- `docs/`: architecture, installation, roadmap, operations.
- `scripts/windows/`: portable/installer packaging helper scripts.
- `scripts/release/`: release checksum/update manifest/signing utilities.

## Hardware Wiring

### Master Arduino (encoders 1-3)

| Encoder | A  | B  | Button |
|---|---|---|---|
| 1 | D2 | D3 | D4 |
| 2 | D5 | D6 | D7 |
| 3 | D8 | D9 | D10 |

### Slave Arduino (encoders 4-5)

| Encoder | A  | B  | Button |
|---|---|---|---|
| 4 | D2 | D3 | D4 |
| 5 | D5 | D6 | D7 |

### I2C between boards

| Master | Slave |
|---|---|
| A4 (SDA) | A4 |
| A5 (SCL) | A5 |
| 5V | 5V |
| GND | GND |

## Serial Event Protocol

Master emits newline-terminated events:
- `V:1` protocol hello emitted on boot
- `E1:+1` encoder 1 clockwise
- `E1:-1` encoder 1 counterclockwise
- `B1:1` encoder 1 button pressed

Parser accepts:
- protocol hello `V:<n>` with positive integer versions
- encoder IDs `1..32`
- button press value only `1` (release/noise values are rejected)

Compatibility rule:
- host currently supports protocol version `1` only
- if firmware announces an incompatible version, host logs mismatch and drops control events
- if no hello is present (older firmware), host keeps legacy behavior for backward compatibility

## Quick Start (Developer)

```bash
cd mama
go test ./...
go run ./cmd/mama-ui
```

Firmware stress validation:

```bash
scripts/firmware/run_encoder_stress_test.sh
scripts/firmware/run_i2c_robustness_test.sh
```

- `run_encoder_stress_test.sh`: validates fast encoder spin + button debounce edge cases.
- `run_i2c_robustness_test.sh`: validates slave I2C packet integrity under burst-load accumulator and button-edge churn.

Then open the shown local URL and use the first-run wizard (or manual controls) to detect the board, test connection, optionally apply a mapping template (streaming/conferencing/music/gaming), map knobs, and save. You can also create a browser-local backup snapshot, restore it, or import/export JSON config files before persisting with **Save Config**. The setup UI supports keyboard-first navigation with visible focus states, announces status updates for assistive technologies, and provides a top-level language selector with initial English/Italian coverage.

Start runtime daemon:

```bash
go run ./cmd/mama
```

## Config Behavior

Config path auto-resolution:
1. `./config.yaml` (portable default for packaged app)
2. `internal/config/default.yaml` (source-repo default)

If no file exists, MAMA creates one automatically.

### Config schema

```yaml
serial:
  port: "COM3"   # Linux example: /dev/ttyACM0
  baud: 115200

debug: true

active_profile: "streaming"

mappings:
  - knob: 1
    target: master_out
    step: 0.02

profiles:
  - name: "streaming"
    mappings:
      - knob: 1
        target: app
        selector:
          kind: exe
          value: "discord.exe"
        step: 0.02

      - knob: 2
        target: group
        selectors:
          - kind: contains
            value: "Game"
          - kind: prefix
            value: "Spotify"
        step: 0.02
```

`target` values:
- `master_out` (implemented)
- `mic_in` (implemented on Windows and Unix hosts with `pactl` or `amixer` available)
- `line_in` (implemented on Windows and Unix hosts with capture tooling available; routed via capture endpoint controls)
- `app` (implemented on Unix hosts with `pactl`; requires `selector`)
- `group` (implemented on Unix hosts with `pactl`; requires `selectors`)

`selector.kind` / `selectors[].kind` values:
- `exact` (exact app/session label match)
- `contains` (substring match)
- `prefix` (prefix match)
- `suffix` (suffix match)
- `glob` (shell-style wildcard, e.g. `*game*`)
- `exe` (executable name match, e.g. `discord.exe`)

Validation rules:
- `serial.port` required
- `serial.baud > 0`
- at least one mapping required (top-level `mappings` or one `profiles[].mappings` set)
- `profiles` entries require unique non-empty `name` values and non-empty `mappings`
- `active_profile` is optional; when omitted and profiles exist, it defaults to the first profile name
- `active_profile` must match one configured profile name when profiles are present
- unique `knob` IDs and `knob > 0`
- `step` in `(0, 1]`
- `app` must define `selector` and must not define `selectors`
- `group` must define `selectors` and must not define `selector`
- selector values must be non-empty strings
- `priority` (optional, `>= 0`) can be set on `app/group` mappings to resolve overlaps; higher priority wins
- overlapping `app/group` mappings with identical precedence score are rejected as ambiguous
- non-`app/group` targets must not define `name`, `selector`, or `selectors`

Compatibility aliases accepted for older configs:
- top-level `port`/`baud` (migrated to `serial.port`/`serial.baud`)
- top-level `knobs` (migrated to `mappings`)
- mapping keys: `id` -> `knob`, `type` -> `target`, `app` -> `name`, `volume_step` -> `step`
- `name` remains accepted for `app/group` mappings as a legacy alias and is migrated to exact-match selector entries

## Continuous Integration

GitHub Actions runs cross-platform tests and security checks on every pull request and on pushes to `main`, using a patched Go toolchain line (`1.24.x`).

Test workflow:
- workflow: `.github/workflows/ci.yml`
- operating systems: `ubuntu-latest`, `windows-latest`, `macos-latest`
- command: `cd mama && go test ./...`

Security workflow:
- workflow: `.github/workflows/security-scan.yml`
- operating system: `ubuntu-latest`
- commands: `cd mama && go mod verify`, `cd mama && go mod tidy && git diff --exit-code -- go.mod go.sum`, and `cd mama && govulncheck ./...`

Issue and pull request templates are available under `.github/ISSUE_TEMPLATE/` and `.github/pull_request_template.md` to standardize triage and review details.

## Release Integrity

Maintainers can generate SHA-256 checksum manifests for packaged artifacts with:

```bash
scripts/release/generate-checksums.sh dist
```

Optional update metadata manifests can be generated with:

```bash
scripts/release/generate-update-manifest.sh <archive-path> <tag> <download-url>
```

Reproducible build guidance is documented in [docs/RELEASE_REPRODUCIBLE_BUILDS.md](docs/RELEASE_REPRODUCIBLE_BUILDS.md).

Release packaging automation (portable + optional installer/update artifacts) is implemented in `.github/workflows/release-artifacts.yml`.

Release signing/notarization automation and maintainer setup details are documented in [docs/SIGNING_AND_NOTARIZATION.md](docs/SIGNING_AND_NOTARIZATION.md) and implemented in `.github/workflows/release-signing.yml`.

## Non-Terminal User Flow

See [docs/INSTALLATION.md](docs/INSTALLATION.md) for packaging and release instructions.

Recommended end-user artifacts:
- `mama.exe` (runtime daemon)
- `mama-ui.exe` (settings GUI)
- `Open Setup UI.cmd` (double-click launcher)
- `Start Mixer.cmd` (double-click launcher)
- `config.yaml` (portable side-by-side config)

This keeps configuration and binaries in one folder and avoids installing system services by default.

## Documentation Index

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/INSTALLATION.md](docs/INSTALLATION.md)
- [docs/ROADMAP.md](docs/ROADMAP.md)
- [docs/RELEASE_REPRODUCIBLE_BUILDS.md](docs/RELEASE_REPRODUCIBLE_BUILDS.md)
- [docs/RELEASE_QA_CHECKLIST.md](docs/RELEASE_QA_CHECKLIST.md)
- [docs/SIGNING_AND_NOTARIZATION.md](docs/SIGNING_AND_NOTARIZATION.md)
- [docs/SUPPORT_POLICY.md](docs/SUPPORT_POLICY.md)
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [SECURITY.md](SECURITY.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT. See `LICENSE`.
