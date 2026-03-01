# Architecture

## System Overview

MAMA is split into 3 runtime components:

1. Hardware input layer (Arduino)
2. Transport/protocol layer (USB serial + text events)
3. Host control layer (Go daemon + setup UI)

## Data Flow

1. User rotates/presses encoder.
2. Master/slave firmware debounces and accumulates quadrature transitions.
3. Master emits serial lines:
   - `V:<version>` once on boot for protocol negotiation
   - `E<id>:+/-<n>`
   - `B<id>:1`
4. Host parser converts line -> typed event and enforces protocol compatibility if version is announced.
5. Mapping table resolves knob ID -> target/action config.
6. Audio backend applies volume change or mute toggle.

## Firmware Split

- `master.ino`
  - reads encoders 1-3 locally
  - polls slave via I2C for encoders 4-5
  - emits unified serial event stream
- `slave.ino`
  - reads encoders 4-5
  - responds to I2C requests with fixed-size packet

Rationale:
- keeps USB connection single-ended on master
- keeps protocol simple and deterministic

## Host App Modules (`mama/internal`)

- `config`
  - YAML load/save
  - schema validation and normalization
  - default config creation for portable mode
- `proto`
  - strict parser for serial line protocol
- `serial`
  - serial open/read loop + port listing
- `audio`
  - platform backend abstraction
  - currently `master_out` only
- `ui`
  - local HTTP setup UI and identify stream endpoint

## Runtime Binaries

- `cmd/mama`
  - runtime daemon
  - reads serial events and applies mapped actions
- `cmd/mama-ui`
  - local setup GUI server
  - config editor + knob identify visualization

## Config Lifecycle

- Path resolution:
  - `./config.yaml` first
  - fallback `internal/config/default.yaml`
- If not found, default file is created.
- UI uses JSON API; persisted as YAML for human-editability.

## Portability and OS Impact

Design target:
- no required system service install
- no required global registry keys
- no auto-start task by default
- no hidden background daemon

Portable package runs from one folder:
- binaries
- launchers
- `config.yaml`

## Known Gaps

- Audio backend does not yet implement:
  - `mic_in`
  - `line_in`
  - `app`
  - `group`
- No signed installer pipeline yet.
- No automated hardware-in-loop test suite yet.
