# Manopola Audio Mixer for Arduino (MAMA)

MAMA is a physical USB audio mixer built with 2 Arduino Nano boards (master + slave) and a Go desktop app.

This repository now includes:
- firmware for 5 rotary encoders with push buttons
- runtime daemon (`mama`) that applies volume actions
- browser-based setup GUI (`mama-ui`) with visual knob identification
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
  - setup UI (`mama-ui`) to edit serial settings/mappings and save config
  - setup UI serial connection test button for validating port + baud before saving
  - setup UI live identify mode (rotate/press a knob and the matching indicator flashes)
- Planned / not yet implemented in audio backend:
  - `mic_in`
  - `line_in`
  - per-app (`app`)
  - app groups (`group`)

## Repository Layout

- `firmware/master/master.ino`: USB-connected master Arduino firmware (encoders 1-3 + I2C poll for 4-5).
- `firmware/slave/slave.ino`: I2C slave Arduino firmware (encoders 4-5).
- `mama/cmd/mama`: runtime daemon (serial events -> audio actions).
- `mama/cmd/mama-ui`: local setup GUI server.
- `mama/internal/*`: config, protocol parser, serial I/O, audio backend, setup UI server.
- `docs/`: architecture, installation, roadmap, operations.
- `scripts/windows/`: release packaging helper scripts.

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
- `E1:+1` encoder 1 clockwise
- `E1:-1` encoder 1 counterclockwise
- `B1:1` encoder 1 button pressed

Parser accepts:
- encoder IDs `1..32`
- button press value only `1` (release/noise values are rejected)

## Quick Start (Developer)

```bash
cd mama
go test ./...
go run ./cmd/mama-ui
```

Then open the shown local URL, set serial port/baud, map knobs, and save.

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

mappings:
  - knob: 1
    target: master_out
    step: 0.02
```

`target` values:
- `master_out` (implemented)
- `mic_in` (planned)
- `line_in` (planned)
- `app` (planned, requires `name`)
- `group` (planned, requires `name`)

Validation rules:
- `serial.port` required
- `serial.baud > 0`
- at least one mapping required
- unique `knob` IDs and `knob > 0`
- `step` in `(0, 1]`
- `name` required for `app/group`, forbidden otherwise

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
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT. See `LICENSE`.
