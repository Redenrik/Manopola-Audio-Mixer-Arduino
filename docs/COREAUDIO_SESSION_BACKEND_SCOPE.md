# CoreAudio Session Backend Scope (macOS)

This document scopes the next implementation step: a dedicated macOS CoreAudio session backend for `app` / `group` targets.

## Objective

Deliver a macOS-native backend that can:

- discover active app audio sessions
- map sessions to stable selector tokens (`exact`, `contains`, `exe`, etc.)
- adjust per-session volume
- toggle per-session mute
- support grouped selectors (`group` target)

## Current Baseline

- macOS now uses an embedded desktop shell window.
- `master_out` works.
- `mic_in` / `line_in` map to default input level.
- `app` / `group` are currently unavailable on macOS when no session backend is present.

## Scope Boundaries

In scope:

- dedicated `darwin` app-session controller implementation
- CoreAudio bridge and session polling/cache
- backend wiring + UI target discovery compatibility
- tests for selector matching, discovery normalization, and volume/mute operations

Out of scope (for this step):

- replacing non-macOS backends
- redesigning mapping schema
- unrelated UI redesign

## Proposed Architecture

### 1) Backend split for session controllers

- Keep Linux `pactl` controller Linux-only.
- Add macOS-specific controller entrypoint:
  - `newAppSessionController()` on `darwin` returns CoreAudio implementation when supported.
  - Fallback remains `nil` when unavailable.

### 2) CoreAudio bridge layer

Create low-level `darwin` bridge package/files for:

- enumerating active process/session objects
- reading per-session volume/mute
- setting per-session volume/mute

Bridge responsibilities:

- isolate cgo/CoreAudio details
- provide Go-friendly structs/errors
- normalize transient CoreAudio errors into retryable/unavailable classes

### 3) Session controller layer

Implement `appSessionController` methods on top of bridge:

- `ListTargets()`
- `ReadState(...)` / `ReadGroupState(...)`
- `Adjust(...)` / `AdjustGroup(...)`
- `ToggleMute(...)` / `ToggleMuteGroup(...)`

Behavior:

- selector matching keeps existing semantics
- group operations dedupe matched sessions by stable session ID
- stale sessions map to `ErrTargetUnavailable`

### 4) Caching and refresh policy

Use a short-lived cache (for example 150-300ms) to avoid expensive bridge calls per encoder event.

Cache content:

- session ID
- display name
- executable/bundle token(s)
- current volume/mute
- timestamp

## Implementation Plan

### Phase 0: Feasibility spike

- validate CoreAudio APIs needed for per-session control on supported macOS versions
- confirm minimum macOS version for reliable session enumeration/control
- produce a compatibility matrix

Exit criteria:

- clear API set + minimum OS decision documented

### Phase 1: Discovery-only backend

- implement bridge discovery
- return discovered app targets via `/api/targets`
- no write operations yet

Exit criteria:

- app sessions appear in UI suggestions on macOS
- selector matching tests pass

### Phase 2: Write operations

- implement per-session adjust/mute
- implement grouped operations
- wire error handling and fallback behavior

Exit criteria:

- `app` and `group` mappings control live sessions on macOS
- no panic/crash on session churn

### Phase 3: Hardening

- debounce/retry for transient CoreAudio failures
- diagnostics logs for session attach/detach
- performance check under rapid encoder input

Exit criteria:

- stable behavior under churn + fast knob turns

## Test Plan

Unit tests:

- selector normalization and match behavior
- session dedupe rules
- group aggregate read-state behavior
- error mapping (`unavailable` vs hard failure)

Integration/smoke:

- discover sessions for at least 3 running apps
- adjust/mute one app repeatedly
- group control across 2+ apps
- app launch/quit churn while rotating knobs quickly

## Risks and Unknowns

- CoreAudio public API support for per-app output sessions may vary by macOS version.
- Session identity stability may change across app restart/output-device switch.
- Permission/security constraints may affect discovery/control on hardened systems.

## Acceptance Criteria

- On macOS, at least one active app session appears in discovered targets when media is playing.
- A knob mapped to `app` changes only that app’s volume.
- A knob mapped to `group` changes all matched app sessions.
- Errors for disappeared sessions are non-fatal and surfaced as target-unavailable.
