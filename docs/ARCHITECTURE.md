# Architecture

## Overview

MAMA is composed of three layers:

1. Hardware input layer (Arduino master/slave encoders)
2. Transport layer (USB serial text protocol)
3. Host control layer (Go app: runtime + local setup UI + platform shell)

The shipped host app is one binary:

- `mama` (`mama.exe` on Windows)

There is no separate runtime/UI executable requirement for end users.

## Runtime Model

- Runtime engine and UI server run in the same process.
- Config changes in the UI are applied live without restart.
- On Windows, desktop mode is enabled by default (embedded window + tray).
- On macOS/Linux, desktop mode runs in browser-shell form (local HTTP UI + runtime loop).
- On macOS/Linux desktop mode includes native tray/menu-bar controls (platform desktop session required).

## Data Flow

1. Physical encoder rotation/press occurs on hardware.
2. Firmware emits serial lines.
3. Host parser converts lines into typed events.
4. Runtime resolves `knob -> mapping`.
5. Audio backend applies volume/mute operations.
6. UI polls runtime state and renders status/health.

## Firmware Topology

- `firmware/master/master.ino`
  - reads local encoders (1-3)
  - polls slave via I2C (encoders 4-5)
  - emits protocol lines over USB serial
- `firmware/slave/slave.ino`
  - reads local encoders (4-5)
  - returns fixed-size I2C packet to master

## Serial Protocol

Primary protocol:

- `MAMA:HELLO:1`
- `E<id>:+/-<n>`
- `B<id>:1`

Compatibility:

- Host supports protocol version `1`.
- Host still accepts legacy hello `V:1`.
- Unsupported hello versions are rejected safely.

## Host Modules

- `mama/cmd/mama`
  - app entrypoint, flags, startup/shutdown orchestration
- `mama/internal/config`
  - YAML schema, migration, validation, defaults
- `mama/internal/proto`
  - serial line parsing and event typing
- `mama/internal/serial`
  - serial open/read/probe/reconnect logic
- `mama/internal/runtime`
  - event loop, retries, logging, metrics helpers
- `mama/internal/mixer`
  - mapping resolution, target control, fallback logic
- `mama/internal/audio`
  - platform audio backends and target discovery
- `mama/internal/ui`
  - HTTP API, startup integration hooks, static UI delivery

## Packaging Model

Recommended release assets:

- Windows installers (`amd64`, `386`)
- macOS universal2 package
- Linux amd64 package

Advanced portable assets are also published for architecture-specific workflows.

## Known Constraints

- Native Windows arm64 build is not officially supported yet.
- Hardware-in-loop validation is still a required manual release gate.
