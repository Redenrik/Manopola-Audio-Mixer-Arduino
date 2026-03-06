# CoreAudio Session Backend Scope (macOS)

This document tracks the macOS `app` / `group` backend after validating the public tap path on a live system.

## Decision

Direct CoreAudio process objects remain discovery-only.

MAMA’s writable macOS path for `app` and `group` targets is a tap-backed mixer built from:

- `AudioHardwareCreateProcessTap`
- a private aggregate device
- a render loop that reads tapped audio, applies software gain/mute, and replays it to the current default output

## Why The Pivot Happened

Live probing on macOS 26.3 showed that:

- process objects are enumerable through `kAudioHardwarePropertyProcessObjectList`
- active apps expose PID, bundle ID, executable path, and output-running state
- those process objects do not expose reliable writable per-app volume or mute controls
- the objects discovered through `kAudioProcessPropertyDevices` resolve to shared device controls, not isolated app controls

That makes direct process-object writes unsuitable for shipping `app` / `group` control.

## Feasibility Gate

Phase 0 is only considered passed if a public-API probe can:

- create a process tap for a running app
- create a private aggregate device
- attach the tap to the aggregate
- start aggregate IO
- observe live audio frames
- re-output the tapped app through software gain/mute

This gate is now met locally when following Apple’s sample sequence:

1. Create the process tap.
2. Create an empty private aggregate device.
3. Add the output subdevice through `kAudioAggregateDevicePropertyFullSubDeviceList`.
4. Add the tap through `kAudioAggregateDevicePropertyTapList`.
5. Wait for aggregate streams to publish.
6. Start `AudioDeviceCreateIOProcID` / `AudioDeviceStart`.

The earlier assumption that the aggregate could be composed in one shot through the create dictionary was the wrong model for this backend.

## Supported Versions

- Minimum writable support: macOS 14.2+
  - `AudioHardwareCreateProcessTap` is only available starting with macOS 14.2.
- Restart resilience improvements: macOS 26.0+
  - use `CATapDescription.bundleIDs`
  - use `CATapDescription.processRestoreEnabled`

## Backend Split

### Discovery bridge

`coreAudioBridge` is discovery-only and is responsible for:

- enumerating process objects
- reading PID
- reading bundle ID
- reading executable path
- reading output-running state

### Tap bridge

The writable path lives in a separate macOS-only tap bridge and is responsible for:

- `EnsureTap(session)`
- `ReleaseTap(sessionKey)`
- `SetGain(sessionKey, percent)`
- `SetMuted(sessionKey, bool)`
- `ReadVirtualState(sessionKey)`
- `Reconcile(processes)`

The tap bridge owns:

- tap creation and destruction
- aggregate device creation and destruction
- aggregate IO start/stop
- software gain/mute state
- output-device rebuilds
- app restart restoration

## Controller Semantics

The Darwin app-session controller now:

- uses process objects only for discovery and selector matching
- uses tap-backed virtual state for `Adjust`, `ToggleMute`, `ReadState`, and group operations
- reports `app` target capabilities based on tap support, not on direct HAL per-process controls

Read state on macOS is therefore virtual mixer state:

- default is `100` / unmuted before a tap is attached
- after attach, MAMA reports the gain/mute values it is applying in the tap render path

## Reconciliation Rules

On macOS 14.2 through 25.x:

- tapped state is preserved in MAMA
- if the app exits, the live tap is torn down
- when the app reappears with the same stable session key, MAMA recreates the tap automatically

On macOS 26.0+:

- taps are created with bundle-ID restore support when a bundle ID exists
- MAMA keeps the tap alive across short app restarts when the output device is unchanged
- output-device changes still force a rebuild because the aggregate path is device-specific

## Diagnostic Command

Non-shipping probe command:

```bash
go run ./cmd/mama-tap-probe -bundle com.spotify.client -duration 5s
```

Useful flags:

- `-volume`
- `-muted`

The command prints the selected process object, output device UID, callback count, and frame count so tap viability can be checked without touching the main app path.

## Shipping Constraints

- public APIs only
- no private CoreAudio APIs
- no unsupported system patches
- no direct process-object volume/mute writes for macOS `app` / `group`

Apple also requires `NSAudioCaptureUsageDescription` for packaged tap capture apps. If MAMA moves to a bundled macOS app distribution path, that permission string must be present in the app bundle metadata.

## Remaining Hardening Work

- permission UX and failure reporting around system audio capture
- underrun / doubled-audio checks under rapid encoder input
- output-device switch soak testing
- multi-app group load testing
- packaging-time audio-capture entitlement/plist review for macOS distribution
