# Support Matrix and Versioning Policy

## 1) Platform Support Matrix

### Desktop host app

| Dimension | Supported | Notes |
|---|---|---|
| OS families | Windows, macOS, Linux | CI validates all three OS families on PRs/pushes. |
| Windows architectures | `amd64`, `386` | Official installers are published for both. |
| macOS architectures | `amd64`, `arm64` | Recommended user package is universal2. |
| Linux architectures | `amd64`, `arm64` | Recommended user package is `amd64`; arm64 portable is available. |
| Windows arm64 | Not yet supported natively | Upstream dependency constraint; use x64 path on ARM devices where available. |
| Go baseline | `1.24.x` (patched) | Matches CI/release workflows. |

### Firmware and protocol compatibility

| Component | Supported baseline | Notes |
|---|---|---|
| Hardware topology | Arduino Nano master + Nano slave | Master via USB, slave via I2C. |
| Protocol hello | `MAMA:HELLO:1` | Host protocol version support: `1`. |
| Legacy hello | `V:1` accepted | Kept for backward compatibility with older firmware. |
| Unsupported protocol versions | Rejected safely | Host logs mismatch and drops control events. |

### Feature-level mapping support

| Mapping target | Status |
|---|---|
| `master_out` | Supported |
| `mic_in` | Supported on hosts with capture endpoint support |
| `line_in` | Supported on hosts with capture endpoint support |
| `app` | Supported on hosts with active per-app session control |
| `group` | Supported on hosts with active session-group control |

## 2) Versioning Policy

MAMA uses Semantic Versioning after `v1.0.0`:
- MAJOR: breaking changes
- MINOR: backward-compatible features
- PATCH: backward-compatible fixes

Before `v1.0.0`, breaking changes are allowed but must be called out in release notes.

## 3) Deprecation Policy

### Config and runtime flags

- Prefer additive changes.
- Keep deprecated aliases for at least one MINOR release.
- Document deprecations in release notes.

### Serial protocol

- New protocol revisions must include explicit compatibility rules.
- Keep previous protocol support for at least one MINOR release after introduction.

### Platform changes

Any support removal requires:
1. advance release-note notice
2. rationale (security/upstream/maintenance)
3. migration path when practical

## 4) Support Window

After `v1.0.0`:
- latest stable: full support
- previous MINOR: security and high-severity fixes
- older lines: best effort
