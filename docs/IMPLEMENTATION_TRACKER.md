# MAMA Implementation Tracker (Execution File)

> Purpose: this is the operational file Codex should follow and update while implementing the not-yet-developed/upgraded parts of the project.

## How to use this file
- Before coding: mark the selected item as `IN_PROGRESS` and add a short implementation plan under it.
- During coding: append progress notes in the `Execution Log` section.
- After coding: mark item `DONE`, link changed files/tests, and note remaining risks.
- Keep updates small and frequent; never batch-update at the end only.

Legend:
- `TODO` = not started
- `IN_PROGRESS` = currently being implemented
- `DONE` = implemented and verified
- `BLOCKED` = cannot continue without external dependency/decision

---

## 1) Core Missing Features

### A. Audio target support
- [x] `DONE` Implement `mic_in` backend support (Windows + Unix strategy decision documented).
  - Implemented:
    - Refactored backend target routing so `master_out` and `mic_in` are controlled by independent volume controllers while preserving existing `master_out` behavior and unsupported handling for other targets.
    - Added Unix microphone controller support using default PulseAudio source commands (`pactl`) with ALSA `amixer` fallback, including volume and mute toggles.
    - Added Windows microphone controller support using Core Audio capture endpoint APIs (`ECapture`) for volume/mute reads and updates.
    - Extended audio backend contract tests to cover `mic_in` adjust/mute behavior, dynamic target discovery when microphone control is available, and unsupported fallback when unavailable.
    - Updated README, troubleshooting, and support-policy docs to reflect the new `mic_in` support status and platform/tooling conditions.
  - Changed files/tests:
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_unix.go`
    - `mama/internal/audio/backend_windows.go`
    - `mama/internal/audio/mic_controller_unix.go`
    - `mama/internal/audio/mic_controller_windows.go`
    - `mama/internal/audio/backend_contract_test.go`
    - `README.md`
    - `docs/TROUBLESHOOTING.md`
    - `docs/SUPPORT_POLICY.md`
- [x] `DONE` Implement `line_in` backend support.
  - Implemented:
    - Extended backend target routing with a dedicated `line_in` controller path so adjust/mute actions are handled independently from `master_out` and `mic_in`.
    - Added platform-specific line-input controller constructors and wired default backend initialization to expose `line_in` when capture endpoint controls are available.
    - Expanded backend contract coverage for `line_in` adjust/mute support, unsupported fallback when controller discovery is unavailable, and target discovery metadata updates.
    - Updated README, troubleshooting, support-policy, and default-config comments to reflect shipped `line_in` support semantics.
  - Changed files/tests:
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_unix.go`
    - `mama/internal/audio/backend_windows.go`
    - `mama/internal/audio/line_controller_unix.go`
    - `mama/internal/audio/line_controller_windows.go`
    - `mama/internal/audio/backend_contract_test.go`
    - `mama/internal/config/config.go`
    - `mama/internal/config/default.yaml`
    - `README.md`
    - `docs/TROUBLESHOOTING.md`
    - `docs/SUPPORT_POLICY.md`
- [ ] `TODO` Implement `app` per-process/session volume control.
- [ ] `TODO` Implement `group` target for grouped app/session controls.
- [x] `DONE` Extend `/api/targets` to expose real discoverable targets (not placeholders).
  - Implemented:
    - Introduced a typed audio discovery model (`audio.DiscoveredTarget`) and migrated backend target listing to return structured metadata (`id`, `type`, `name`).
    - Updated `/api/targets` to return `discovered` from backend enumeration while preserving backward-compatible `known` and `supported` fields.
    - Added tests for backend discovery contracts and setup UI target endpoint response shape.
    - Updated README feature notes to document the expanded `/api/targets` response.
  - Changed files/tests:
    - `mama/internal/audio/audio.go`
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_contract_test.go`
    - `mama/internal/ui/server.go`
    - `mama/internal/ui/server_targets_test.go`
    - `mama/cmd/mama/main_integration_test.go`
    - `README.md`

### B. Config and mapping model maturity
- [x] `DONE` Define and validate robust schema for `app/group` selectors (exact match, wildcard, executable name, etc.).
  - Implemented:
    - Added explicit mapping selector schema (`selector` for `app`, `selectors` for `group`) with normalized selector kinds: `exact`, `contains`, `prefix`, `suffix`, `glob`, and `exe`.
    - Added validation enforcing per-target selector shape, non-empty selector values, and selector-kind allowlist while preserving backward compatibility via legacy `name` migration to exact selectors.
    - Expanded config/docs coverage with schema examples and troubleshooting updates.
  - Changed files/tests:
    - `mama/internal/config/config.go`
    - `mama/internal/config/config_test.go`
    - `README.md`
    - `docs/TROUBLESHOOTING.md`
- [x] `DONE` Add conflict/precedence rules for overlapping mappings.
  - Implemented:
    - Added optional per-mapping `priority` support for `app/group` targets to provide explicit overlap precedence while preserving backward compatibility for existing configs.
    - Added overlap validation that detects ambiguous `app/group` selector collisions and rejects ties when precedence scores are identical.
    - Added specificity-based precedence scoring (`priority` first, selector specificity second) as deterministic conflict resolution rules.
    - Documented precedence and troubleshooting guidance for overlap/priority validation failures.
  - Changed files/tests:
    - `mama/internal/config/config.go`
    - `mama/internal/config/config_test.go`
    - `README.md`
    - `docs/TROUBLESHOOTING.md`
- [x] `DONE` Add profile support (multiple mapping sets with active profile selection).
  - Implemented:
    - Extended config schema with `profiles[]` and `active_profile`, including validation for unique profile names, non-empty profile mappings, and active-profile existence.
    - Added active-profile mapping resolution so runtime knob lookups use the selected profile while preserving backward compatibility for top-level `mappings`.
    - Added unit tests for profile loading/validation/defaulting and updated README config schema/validation docs.
  - Changed files/tests:
    - `mama/internal/config/config.go`
    - `mama/internal/config/config_test.go`
    - `README.md`

---

## 2) Reliability / Protocol / Runtime Hardening

- [x] `DONE` Add protocol versioning and compatibility checks between firmware and host.
  - Implemented:
    - Added `V:<version>` protocol hello event support in firmware and host parser.
    - Added host compatibility gating to drop control events when firmware announces an unsupported protocol version, with clear logs.
    - Added parser/compatibility tests and updated protocol docs.
- [x] `DONE` Implement reconnect strategy with bounded backoff and clear state transitions.
  - Implemented:
    - Added reusable bounded exponential backoff helper with unit tests for capping/reset/default behavior.
    - Refactored host serial runtime to continuously reconnect after open/read failures until shutdown signal/context cancellation.
    - Added explicit serial state transition logs (`connecting`, `connected`, `disconnected`, `reconnecting`) with retry delay details.
  - Changed files/tests:
    - `mama/cmd/mama/main.go`
    - `mama/internal/runtime/backoff.go`
    - `mama/internal/runtime/backoff_test.go`
- [x] `DONE` Add runtime metrics/counters (parse errors, dropped events, reconnect count, backend failures).
  - Implemented:
    - Added `internal/runtime` metrics collector with atomic counters and deterministic snapshot formatting.
    - Wired counters into host reconnect/session code paths for parse errors, dropped events, reconnect attempts, and backend failures.
    - Added runtime metrics summary log output on reconnect and session close.
  - Changed files/tests:
    - `mama/internal/runtime/metrics.go`
    - `mama/internal/runtime/metrics_test.go`
    - `mama/cmd/mama/main.go`
    - `docs/TROUBLESHOOTING.md`
- [x] `DONE` Add deterministic structured logs for troubleshooting.
  - Implemented:
    - Added a shared structured logging helper that emits deterministic `event=<name> key=value` entries with stable sorted field ordering.
    - Converted host runtime startup/serial/protocol/debug log points to structured events while preserving runtime signal and metrics visibility.
    - Added formatter unit tests and updated troubleshooting guidance with the new log format and metrics event name.
  - Changed files/tests:
    - `mama/internal/runtime/structured_log.go`
    - `mama/internal/runtime/structured_log_test.go`
    - `mama/cmd/mama/main.go`
    - `docs/TROUBLESHOOTING.md`
- [x] `DONE` Add long-run soak test plan and scripted verification artifacts.
  - Implemented:
    - Added a dedicated long-run soak test plan with scope, cadence, and artifact review criteria for host runtime verification.
    - Added a reusable soak script that runs repeated host checks, writes timestamped artifacts, and fails fast on regressions.
  - Changed files/tests:
    - `docs/SOAK_TEST_PLAN.md`
    - `scripts/soak/run_host_soak.sh`

---

## 3) Test Coverage Expansion

### A. Host-side tests
- [x] `DONE` Add integration tests with recorded serial fixtures (valid bursts + malformed lines + disconnect cases).
  - Implemented:
    - Added fixture-driven host runtime integration tests that replay recorded serial event sequences.
    - Covered valid + malformed mixed bursts and disconnect termination behavior with explicit assertions on backend calls.
    - Verified runtime metrics side effects (parse error accounting) under fixture replay.
  - Changed files/tests:
    - `mama/cmd/mama/main.go`
    - `mama/cmd/mama/main_integration_test.go`
    - `mama/cmd/mama/testdata/serial_fixtures/mixed_burst.txt`
    - `mama/cmd/mama/testdata/serial_fixtures/disconnect_after_events.txt`
- [x] `DONE` Add backend contract tests for each supported target type.
  - Implemented:
    - Refactored audio backend target handling into shared `baseBackend` logic with a mockable volume controller for deterministic contract testing.
    - Added backend contract tests for supported `master_out` behavior (adjust, clamp, mute toggle, target listing) and explicit unsupported-target rejection (`mic_in`, `line_in`, `app`, `group`).
    - Added error propagation assertions for backend-to-volume adapter interactions.
  - Changed files/tests:
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_unix.go`
    - `mama/internal/audio/backend_windows.go`
    - `mama/internal/audio/backend_contract_test.go`
- [x] `DONE` Add config migration/compatibility tests for future schema evolution.
  - Implemented:
    - Added config parsing compatibility shims for legacy aliases (`port`/`baud`, `knobs`, `id`, `type`, `app`, `volume_step`).
    - Added migration/precedence tests to ensure legacy configs load while current schema keys remain authoritative when both are present.
    - Documented accepted compatibility aliases in README config docs.
  - Changed files/tests:
    - `mama/internal/config/config.go`
    - `mama/internal/config/config_test.go`
    - `README.md`

### B. Firmware-side validation
- [ ] `TODO` Add stress tests for fast encoder spin handling and button debounce edge cases.
- [ ] `TODO` Add I2C robustness checks for master/slave packet integrity under load.

---

## 4) Setup UI and End-User Experience

- [ ] `TODO` Add first-run guided wizard (detect board -> test port -> map knobs -> save -> verify).
- [ ] `TODO` Add mapping templates (streaming/conferencing/music/gaming).
- [ ] `TODO` Add config backup/restore/import/export from UI.
- [ ] `TODO` Add localization framework and initial EN/IT coverage.
- [ ] `TODO` Add accessibility pass (keyboard navigation, labels, contrast, status clarity).

---

## 5) Release Engineering and Distribution

- [x] `DONE` Add CI matrix builds (Windows/Linux/macOS) with test execution.
  - Implemented:
    - Added a GitHub Actions CI workflow for pushes to `main` and all pull requests with a three-OS matrix (`ubuntu-latest`, `windows-latest`, `macos-latest`).
    - Configured each matrix run to execute `go test ./...` from the `mama/` module using Go `1.22.x` with dependency caching.
    - Documented CI behavior and scope in `README.md` for contributors.
  - Changed files/tests:
    - `.github/workflows/ci.yml`
    - `README.md`
- [x] `DONE` Add release artifact checksums and reproducible-build notes.
  - Implemented:
    - Added a reusable release checksum script that generates deterministic SHA-256 manifests for packaged artifacts with `sha256sum`/`shasum` fallback support.
    - Added a reproducible-build guidance document covering release prerequisites, deterministic build flow, checksum generation, and verification commands.
    - Updated maintainer-facing installation and repository docs to include checksum generation and reproducible-build references.
  - Changed files/tests:
    - `scripts/release/generate-checksums.sh`
    - `docs/RELEASE_REPRODUCIBLE_BUILDS.md`
    - `docs/INSTALLATION.md`
    - `README.md`
- [ ] `TODO` Add signing/notarization workflow docs and automation.
- [ ] `TODO` Add optional installer/update path while preserving portable mode.
- [ ] `TODO` Add automated changelog/release-notes generation.

---

## 6) Security, Governance, and v1.0 Readiness

- [x] `DONE` Add `SECURITY.md` with disclosure workflow.
  - Implemented:
    - Added a root-level `SECURITY.md` that defines supported-version expectations for the current pre-`v1.0` phase and future policy updates.
    - Documented private vulnerability reporting channels, required report contents, triage/response timelines, and coordinated disclosure steps.
    - Added a `README.md` documentation index link so the security policy is easy to discover.
  - Changed files/tests:
    - `SECURITY.md`
    - `README.md`
- [x] `DONE` Add dependency and vulnerability scanning in CI.
  - Implemented:
    - Added a dedicated GitHub Actions security workflow with two jobs: dependency audit (`go mod verify` + tidy drift check) and `govulncheck` scanning for the `mama/` module.
    - Configured scans to run on pushes to `main` and all pull requests with deterministic Go setup (`1.24.x`) and module-scoped working directory.
    - Documented the new security scanning workflow and commands in `README.md`.
  - Changed files/tests:
    - `.github/workflows/security-scan.yml`
    - `README.md`
- [x] `DONE` Define support matrix and deprecation/versioning policy.
  - Implemented:
    - Added a dedicated support policy document that defines host OS/architecture support, firmware/protocol compatibility expectations, and feature-level target support status.
    - Defined semantic versioning rules (post-`v1.0.0`), pre-`v1.0.0` breaking-change expectations, and deprecation timelines for config, protocol, and platform support.
    - Added release checklist hooks and linked the policy from contributor-facing docs.
  - Changed files/tests:
    - `docs/SUPPORT_POLICY.md`
    - `README.md`
    - `CONTRIBUTING.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add issue/PR templates and release QA checklist.
  - Implemented:
    - Added GitHub issue templates for bug reports and feature requests plus issue-config defaults that disable blank issues and direct security reports to private disclosure.
    - Added a pull request template that captures summary, scope, validation, compatibility/risk, and release-note notes.
    - Added a maintainer release QA checklist and linked template/checklist usage from contributor-facing docs.
  - Changed files/tests:
    - `.github/ISSUE_TEMPLATE/bug_report.yml`
    - `.github/ISSUE_TEMPLATE/feature_request.yml`
    - `.github/ISSUE_TEMPLATE/config.yml`
    - `.github/pull_request_template.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `README.md`
    - `CONTRIBUTING.md`
- [ ] `TODO` Run v1.0 readiness review and publish acceptance criteria.

---

## Execution Log

- 2026-03-01: Tracker created from roadmap gaps; all items initialized as `TODO`.
- 2026-03-01: Implemented protocol versioning handshake (`V:1`) and host compatibility checks. Updated firmware boot output, host parser/runtime gating, setup UI event labeling, and parser tests. Verified with `go test ./...`.
- 2026-03-01: Implemented serial reconnect strategy with bounded exponential backoff and clear runtime state logs in host runtime. Added dedicated backoff unit tests and verified with `go test ./...`.
- 2026-03-01: Implemented host runtime metrics counters (parse errors, dropped events, reconnect count, backend failures), integrated counter updates into reconnect/session paths, and added metrics snapshot logging plus unit tests. Verified with `go test ./...`.
- 2026-03-02: Implemented deterministic structured runtime logging (`event=<name> key=value`), migrated host runtime log emitters to structured events, added formatter unit tests, and updated troubleshooting docs. Verified with `go test ./...`.
- 2026-03-02: Added long-run soak verification plan and scripted artifacts (`scripts/soak/run_host_soak.sh`) with timestamped outputs under `artifacts/soak/`. Verified with `go test ./...` and `scripts/soak/run_host_soak.sh 3`.
- 2026-03-02: Added host integration tests with recorded serial fixtures for mixed valid/malformed bursts and disconnect handling. Refactored session loop to support fixture-driven channel replay while preserving runtime behavior. Verified with `go test ./...`.
- 2026-03-02: Added audio backend contract coverage for all currently supported/declared target types by extracting shared backend logic behind a mockable volume controller and adding deterministic tests for `master_out` semantics, unsupported target rejection, and error propagation. Verified with `go test ./...`.
- 2026-03-02: Added config schema migration compatibility for legacy aliases (top-level `port`/`baud`, `knobs`, and mapping alias keys), plus tests covering migration and current-schema precedence; documented aliases in README. Verified with `go test ./...`.
- 2026-03-02: Added GitHub Actions CI matrix testing across Linux/Windows/macOS for the `mama` Go module, running `go test ./...` on push/PR, and documented the workflow in README. Verified locally with `cd mama && go test ./...`.

- 2026-03-02: Extended `/api/targets` with structured backend discovery (`discovered` with id/type/name), kept backward-compatible `known`/`supported` fields, added UI endpoint tests, and updated backend discovery contracts. Verified with `cd mama && go test ./...`.

- 2026-03-02: Added CI security scanning workflow (`.github/workflows/security-scan.yml`) covering dependency verification (`go mod verify` + tidy drift) and vulnerability scanning (`govulncheck ./...`), plus README CI documentation updates. Verified locally with `cd mama && go test ./...` and `cd mama && go mod verify`.

- 2026-03-02: Remediated CI vulnerability-scan false failures caused by scanning with an out-of-date Go stdlib baseline by upgrading CI/security workflows from Go `1.22.x` to `1.24.x`; this aligns `govulncheck` with patched standard-library fixes while preserving module compatibility checks. Verified locally with `cd mama && go test ./...` and `cd mama && go mod verify` (network-limited environment prevented local `govulncheck` install).

- 2026-03-02: Added release checksum generation (`scripts/release/generate-checksums.sh`) and reproducible-build guidance (`docs/RELEASE_REPRODUCIBLE_BUILDS.md`), then documented maintainer checksum steps in installation/README docs. Verified with `scripts/release/generate-checksums.sh <temp_artifacts>` + `sha256sum -c`, and `cd mama && go test ./...`.
- 2026-03-02: Added a project `SECURITY.md` with supported-version guidance, private disclosure channels, and response SLAs; linked it from the README documentation index for contributor visibility. Verified with `cd mama && go test ./...`.
- 2026-03-02: Defined and published a formal support matrix + deprecation/versioning policy in `docs/SUPPORT_POLICY.md`, linked it from README and CONTRIBUTING for discoverability, and completed governance tracker item updates. Verified with `cd mama && go test ./...`.

- 2026-03-02: Added issue and PR templates for standardized triage/review, published a release QA checklist for pre-tag validation, and linked the new governance assets from README/CONTRIBUTING. Verified with `cd mama && go test ./...`.
- 2026-03-02: Implemented robust `app/group` selector schema validation (`selector`/`selectors` with exact/contains/prefix/suffix/glob/exe kinds), added backward-compatible legacy `name` migration behavior, expanded config tests for valid/invalid selector paths, and updated README + troubleshooting docs. Verified with `cd mama && go test ./...`.
- 2026-03-02: Added `app/group` overlap conflict rules with deterministic precedence (`priority` + selector specificity), rejecting ambiguous precedence ties, plus config/docs/test updates for priority validation and overlap handling. Verified with `cd mama && go test ./...`.
- 2026-03-02: Implemented config profile support (`profiles` + `active_profile`) with backward-compatible top-level mappings fallback, active-profile runtime mapping resolution, and validation for profile naming/selection; added profile-focused config tests and README schema/validation updates. Verified with `cd mama && go test ./...`.
- 2026-03-02: Implemented `mic_in` backend support with platform-specific strategy (`pactl`/`amixer` on Unix, Core Audio capture endpoint on Windows), refactored backend controller routing, expanded audio contract tests for `mic_in` + discovery behavior, and updated support/troubleshooting docs. Verified with `cd mama && go test ./internal/audio ./internal/ui ./cmd/mama` and `cd mama && go test ./...`.

- 2026-03-02: Implemented `line_in` backend support by adding dedicated backend routing/controller wiring (Unix + Windows capture tooling path), extending audio backend contract tests for `line_in` adjust/mute/discovery behavior, and updating user-facing docs/support matrix to mark `line_in` as supported where capture controls are available. Verified with `cd mama && go test ./internal/audio ./...`.


---

## Copy/Paste Prompt for Codex (No Modification Needed)

Use this exact prompt in a new Codex run:

```text
Read and follow `docs/IMPLEMENTATION_TRACKER.md` as the source of truth.

Task:
1) Pick the highest-impact `TODO` item not blocked.
2) Mark it `IN_PROGRESS` in the tracker and add a short implementation plan under that item.
3) Implement the change end-to-end (code + tests + docs as needed).
4) Run relevant checks/tests.
5) Mark the item `DONE` (or `BLOCKED` with reason), and update the `Execution Log` with what was shipped.
6) Commit with a focused message and prepare a PR summary.

Constraints:
- Do not work on already `DONE` items.
- Keep changes scoped to one logical item per iteration.
- If you discover prerequisites, add them as new checklist entries before continuing.
- Preserve backward compatibility unless the tracker item explicitly says otherwise.
```
