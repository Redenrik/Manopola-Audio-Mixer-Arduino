# Manopola Audio Mixer for Arduino (MAMA)

MAMA is a USB rotary audio mixer project built with two Arduino Nano boards and a Go daemon.

## Current Status

- Hardware protocol is implemented for 5 encoders + buttons.
- Go daemon reads serial events, maps knobs from YAML, and controls system master volume.
- Currently implemented audio backend action: `master_out` adjust + mute toggle.
- Target types `mic_in`, `line_in`, `app`, and `group` are defined in config, but not implemented in backends yet.

## Repository Layout

- `firmware/master/master.ino`: USB-connected master Arduino firmware (encoders 1-3 + I2C poll).
- `firmware/slave/slave.ino`: I2C slave Arduino firmware (encoders 4-5).
- `mama/`: Go desktop daemon.

## Hardware

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

### I2C wiring

| Master | Slave |
|---|---|
| A4 (SDA) | A4 |
| A5 (SCL) | A5 |
| 5V | 5V |
| GND | GND |

## Serial Protocol

The master sends newline-terminated events to the PC:

- `E1:+1` encoder 1 clockwise
- `E1:-1` encoder 1 counterclockwise
- `B1:1` button 1 pressed

## Desktop Daemon

### Build and run

```bash
cd mama
go mod tidy
go run ./cmd/mama -config internal/config/default.yaml
```

### Default config

`mama/internal/config/default.yaml` is set to `master_out` on all 5 knobs so first run works without unsupported-target errors.

## Config Format

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

- `serial.port` is required.
- At least one mapping is required.
- `knob` IDs must be unique and > 0.
- `step` must be in `(0, 1]`.
- `name` is required for `app`/`group`; forbidden for other targets.

## Notes

- On Ctrl+C, the daemon now closes the serial port and exits cleanly.
- Button events only accept `B<id>:1`; release/noise values are rejected.

## License

MIT. See `LICENSE`.
