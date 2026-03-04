# Support Matrix and Versioning Policy

This document defines what MAMA versions are supported, what environments are expected to work, and how breaking changes are introduced.

## 1) Support Matrix

### Desktop host (`mama`, `mama-ui`)

| Dimension | Supported | Notes |
|---|---|---|
| Operating systems | Windows, Linux, macOS | CI validates all three OS families on each PR/push. |
| CPU architecture | `amd64`, `arm64` | Recommended assets are OS-level; advanced packages publish `amd64` and `arm64` for Linux/macOS, while Windows currently ships `amd64` (native `arm64` pending dependency support). |
| Go toolchain (contributors/CI) | Latest patched `1.24.x` | Matches the CI baseline and vulnerability scan expectations. |

### Firmware and protocol compatibility

| Component | Supported baseline | Notes |
|---|---|---|
| Hardware topology | Arduino Nano master + Arduino Nano slave | Master connected via USB; slave connected over I2C. |
| Serial protocol hello | `MAMA:HELLO:1` | Host currently supports protocol version `1`; legacy `V:1` remains accepted. |
| Host behavior for missing hello | Compatible | Host keeps legacy compatibility if firmware does not emit a protocol hello line. |
| Host behavior for unsupported hello | Safe reject | Host logs incompatibility and drops control events. |

### Feature-level support status

| Mapping target | Status |
|---|---|
| `master_out` | Supported |
| `mic_in` | Supported on Windows + Unix hosts with capture endpoint tooling available |
| `line_in` | Supported on Windows + Unix hosts with capture endpoint tooling available |
| `app` | Supported on Windows + Unix hosts with active per-app session control support available |
| `group` | Supported on Windows + Unix hosts with active session-group control support available |

## 2) Versioning Policy

MAMA follows **Semantic Versioning** once `v1.0.0` is released:

- **MAJOR** (`X.0.0`): incompatible API/config/protocol changes.
- **MINOR** (`x.Y.0`): backward-compatible feature additions.
- **PATCH** (`x.y.Z`): backward-compatible bug/security fixes.

Until `v1.0.0`, breaking changes may still happen, but must be clearly called out in release notes and migration docs.

## 3) Deprecation Policy

### Config schema and runtime flags

- New config keys should be additive when possible.
- If a key is superseded, keep legacy aliases for **at least one MINOR release** after the replacement ships.
- Emit clear logs/docs notes when deprecated config forms are encountered.
- Remove deprecated config behavior only in a MAJOR release (or in pre-`v1.0.0` with explicit migration guidance).

### Serial protocol

- Protocol changes must include a bumped protocol version and compatibility rules.
- Host support should maintain the prior protocol version for **at least one MINOR release** after introducing a new protocol version.
- Protocol removals require explicit release-note warnings and test coverage updates.

### Platform support

- OS or architecture support removals require:
  1. advance notice in release notes,
  2. tracker/roadmap updates,
  3. a clear rationale (upstream breakage, security, or maintenance burden).

## 4) Support Window Expectations

After `v1.0.0`:

- Latest stable release: full support (features + fixes).
- Previous MINOR line: security and high-severity bug fixes only.
- Older lines: best effort / community support.

Before `v1.0.0`, the project remains best-effort with emphasis on current `main` and latest tagged pre-release.

## 5) Maintainer Release Checklist Hooks

For each release, maintainers should confirm:

1. CI passes on supported OS matrix.
2. Security scan workflow passes.
3. Any deprecations are documented in release notes.
4. Any compatibility-impacting changes include migration instructions.
