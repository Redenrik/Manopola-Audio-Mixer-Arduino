# Architecture

## Overview

MAMA has three layers:

1. Hardware input layer (Arduino master/slave)
2. Transport layer (USB serial text protocol)
3. Host control layer (Go desktop app + audio backend)

The shipped desktop product is a single app process (`mama`) that includes:
- runtime event loop
- setup API and embedded UI page
- tray behavior and background mode

## Data Flow

1. User rotates or presses a physical encoder.
2. Firmware debounces and emits serial lines.
3. Host parser converts lines into typed events.
4. Runtime resolves knob -> mapping target.
5. Audio backend applies volume/mute action.
6. UI polls/streams state and reflects live status.

## Firmware Topology

- `firmware/master/master.ino`
  - local read for encoders 1-3
  - I2C poll for slave encoders 4-5
  - USB serial output to host
- `firmware/slave/slave.ino`
  - local read for encoders 4-5
  - fixed-size I2C response packet to master

## Serial Protocol

Hello and control events:
- `MAMA:HELLO:1`
- `E<id>:+/-<n>`
- `B<id>:1`

Compatibility:
- protocol version `1` is supported
- legacy `V:<n>` remains accepted for older firmware
- unsupported announced versions are rejected safely

## Host Modules

- `mama/internal/config`
  - load/save YAML
  - schema validation and normalization
- `mama/internal/proto`
  - serial protocol parser
- `mama/internal/serial`
  - port open/read/reconnect behavior
- `mama/internal/audio`
  - platform-specific audio target control
- `mama/internal/mixer`
  - target resolution, grouping, conflict handling
- `mama/internal/runtime`
  - event loop, mapping execution, metrics/logging
- `mama/internal/ui`
  - setup HTTP endpoints and embedded UI

## Runtime Binaries

- `mama/cmd/mama`
  - production app binary (runtime + setup UI + tray mode)
- `mama/cmd/mama-ui`
  - compatibility/developer-only UI server entrypoint (not required for end users)

## Packaging Model

Release flow publishes easy user artifacts:
- Windows installers (64-bit and 32-bit)
- macOS universal package
- Linux package

Advanced architecture-specific portable artifacts are also published for power users.

## Current Constraints

- native Windows arm64 build is blocked by upstream dependency compatibility
- hardware-in-loop validation remains a maintainer/manual gate
