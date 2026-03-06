# Support Matrix and Versioning Policy

## 1) Platform Support Matrix

### Desktop host app

| Dimension | Supported | Notes |
|---|---|---|
| OS families | Windows, macOS, Linux | CI validates all three OS families. |
| Windows architectures | `amd64`, `386` | Official installers are published for both. |
| macOS architectures | `amd64`, `arm64` | Recommended package is universal2. |
| Linux architectures | `amd64`, `arm64` | Recommended user package is `amd64`; arm64 portable is advanced asset. |
| Windows arm64 | Not officially supported natively | Use x64 compatibility path where available. |
| Go baseline | `1.22+` | CI/release workflows currently use `1.24.x`. |

### UX/platform integration

| Capability | Windows | macOS | Linux |
|---|---|---|---|
| Desktop shell mode | Yes (embedded shell) | Yes (browser shell) | Yes (browser shell) |
| Tray icon / close-to-tray UX | Yes | Yes (menu bar) | Yes (status tray; DE dependent) |
| Browser-based local UI | Optional (`-desktop=false`) | Yes | Yes |
| In-app startup integration | Yes | Yes | Yes (XDG autostart) |

### Firmware and protocol compatibility

| Component | Supported baseline | Notes |
|---|---|---|
| Hardware topology | Arduino Nano master + Nano slave | Master over USB, slave over I2C. |
| Protocol hello | `MAMA:HELLO:1` | Host protocol support: version `1`. |
| Legacy hello | `V:1` accepted | Backward compatibility path remains enabled. |
| Unsupported versions | Rejected safely | Host logs mismatch and ignores control events. |

### Feature-level mapping support

| Mapping target | Support notes |
|---|---|
| `master_out` | Supported on all platforms |
| `mic_in` | Supported when capture endpoint backend exists |
| `line_in` | Supported when capture endpoint backend exists |
| `app` | Supported when per-app session backend is available and session is active |
| `group` | Supported when per-app session backend is available and matching sessions are active |

## 2) Versioning Policy

MAMA uses Semantic Versioning after `v1.0.0`:

- MAJOR: breaking changes
- MINOR: backward-compatible features
- PATCH: backward-compatible fixes

Before `v1.0.0`, breaking changes are allowed but must be explicitly called out in release notes.

## 3) Deprecation Policy

### Config schema and runtime flags

- Prefer additive changes over replacements.
- Keep deprecated aliases for at least one MINOR release.
- Document deprecations and migration path in release notes.

### Serial protocol

- New protocol revisions must define explicit compatibility behavior.
- Keep prior protocol support for at least one MINOR release after introducing a new protocol line.

### Platform support changes

Any support removal requires:

1. advance release-note notice
2. rationale (security/upstream/maintenance)
3. migration path where practical

## 4) Support Window

After `v1.0.0`:

- latest stable: full support
- previous MINOR line: security and high-severity fixes
- older lines: best effort
