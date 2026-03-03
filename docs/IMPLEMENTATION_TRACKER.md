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
- [x] `DONE` Implement `app` per-process/session volume control.
  - Implemented:
    - Added backend `app` target routing via a dedicated session-controller abstraction while preserving existing `master_out`/`mic_in`/`line_in` behavior and unsupported handling when app session tooling is unavailable.
    - Implemented Unix per-app/session volume + mute controls using `pactl` sink-input discovery and selector-driven matching (`exact`, `contains`, `prefix`, `suffix`, `glob`, `exe`).
    - Wired runtime mapping execution to forward app selector context to the backend, added backend + parser/controller tests for app target behavior, and updated support/docs to reflect shipped `app` support scope.
  - Changed files/tests:
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_unix.go`
    - `mama/internal/audio/backend_windows.go`
    - `mama/internal/audio/app_controller_unix.go`
    - `mama/internal/audio/app_controller_windows.go`
    - `mama/internal/audio/backend_contract_test.go`
    - `mama/internal/audio/app_controller_unix_test.go`
    - `mama/cmd/mama/main.go`
    - `mama/cmd/mama/main_integration_test.go`
    - `mama/internal/config/config.go`
    - `mama/internal/config/default.yaml`
    - `README.md`
    - `docs/TROUBLESHOOTING.md`
    - `docs/SUPPORT_POLICY.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Implement `group` target for grouped app/session controls.
  - Implemented:
    - Extended backend routing so `group` mappings dispatch to dedicated grouped session operations while preserving backward-compatible behavior for existing targets.
    - Added Unix grouped app/session volume + mute control using `pactl` sink-input discovery, multi-selector matching, and de-duplicated per-session command application.
    - Wired runtime mapping token serialization for `group` selectors, expanded backend/controller/runtime tests, and updated support/config/troubleshooting documentation for shipped scope.
  - Changed files/tests:
    - `mama/internal/audio/backend_core.go`
    - `mama/internal/audio/backend_contract_test.go`
    - `mama/internal/audio/app_controller_unix.go`
    - `mama/internal/audio/app_controller_unix_test.go`
    - `mama/internal/audio/app_controller_windows.go`
    - `mama/cmd/mama/main.go`
    - `mama/cmd/mama/main_integration_test.go`
    - `mama/internal/config/config.go`
    - `mama/internal/config/default.yaml`
    - `README.md`
    - `docs/SUPPORT_POLICY.md`
    - `docs/TROUBLESHOOTING.md`
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
- [x] `DONE` Add stress tests for fast encoder spin handling and button debounce edge cases.
  - Implemented:
    - Extracted reusable master encoder state logic into `firmware/master/encoder_logic.h`, and wired `master.ino` to use the shared helpers so runtime behavior and stress tests execute the same code path.
    - Added a native firmware stress test (`firmware/master/tests/encoder_logic_stress_test.cpp`) covering rapid multi-detent quadrature bursts and button bounce/re-press debounce edge cases.
    - Added a runnable validation script (`scripts/firmware/run_encoder_stress_test.sh`) and documented usage in README for repeatable firmware-side verification.
  - Changed files/tests:
    - `firmware/master/master.ino`
    - `firmware/master/encoder_logic.h`
    - `firmware/master/tests/encoder_logic_stress_test.cpp`
    - `scripts/firmware/run_encoder_stress_test.sh`
    - `README.md`
- [x] `DONE` Add I2C robustness checks for master/slave packet integrity under load.
  - Implemented:
    - Extracted slave I2C packet-build/consume behavior into `firmware/slave/i2c_packet.h` so runtime ISR logic and tests share the same packet accounting semantics.
    - Added a native robustness test (`firmware/slave/tests/i2c_packet_integrity_test.cpp`) that validates fixed packet width, clamp/carry-over behavior under burst load, and press-edge reset semantics across repeated request cycles.
    - Added a runnable validation script and README guidance for repeatable I2C integrity checks in firmware verification workflows.
  - Changed files/tests:
    - `firmware/slave/slave.ino`
    - `firmware/slave/i2c_packet.h`
    - `firmware/slave/tests/i2c_packet_integrity_test.cpp`
    - `scripts/firmware/run_i2c_robustness_test.sh`
    - `README.md`

---

## 4) Setup UI and End-User Experience

- [x] `DONE` Add first-run guided wizard (detect board -> test port -> map knobs -> save -> verify).
  - Implemented:
    - Added a dedicated first-run wizard panel in the setup UI with explicit detect/test/map/save/verify steps and reset/next progression controls.
    - Wired wizard status updates to existing refresh-port, port-test, identify, and save actions so guided flow stays aligned with manual setup behavior.
    - Added UI server coverage that asserts wizard controls are present in the embedded index page and documented the new guided flow in README.
  - Changed files/tests:
    - `mama/internal/ui/static/index.html`
    - `mama/internal/ui/server_test.go`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add mapping templates (streaming/conferencing/music/gaming).
  - Implemented:
    - Added four built-in mapping template presets in setup UI (streaming, conferencing, music, gaming) with one-click apply behavior that hydrates the editable mappings grid.
    - Added template selector + apply controls to the mappings panel while preserving existing manual mapping edits and save flow.
    - Extended setup UI server index coverage and README feature/setup docs to include the new template workflow.
  - Changed files/tests:
    - `mama/internal/ui/static/index.html`
    - `mama/internal/ui/server_test.go`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add config backup/restore/import/export from UI.
  - Implemented:
    - Added setup UI controls for browser-local backup snapshots (`Backup Snapshot`/`Restore Snapshot`) plus JSON config export/import actions.
    - Implemented form-state serialization/apply helpers so backup/restore/import/export reuse the same config payload shape as Save Config while keeping existing save behavior backward-compatible.
    - Expanded setup UI index-page server test assertions and README setup/feature docs to include the backup/restore/import/export workflow.
  - Changed files/tests:
    - `mama/internal/ui/static/index.html`
    - `mama/internal/ui/server_test.go`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add localization framework and initial EN/IT coverage.
  - Implemented:
    - Added a lightweight setup-UI localization dictionary with English/Italian translations for page labels, wizard content, template descriptions, and runtime status text.
    - Added a persisted language selector in the setup UI that re-renders static/dynamic content (`wizard`, template options, knob hints) without changing API request/response formats.
    - Expanded setup UI server assertions for localization artifacts and documented the EN/IT language-toggle behavior in README.
  - Changed files/tests:
    - `mama/internal/ui/static/index.html`
    - `mama/internal/ui/server_test.go`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add accessibility pass (keyboard navigation, labels, contrast, status clarity).
  - Implemented:
    - Improved setup UI keyboard accessibility with explicit `:focus-visible` styles for all form controls/buttons and updated status contrast tokens for clearer state visibility.
    - Added accessibility metadata in the setup page: ARIA labels on key controls, `role="status"` live regions for dynamic status messages, and assertive error announcement behavior.
    - Updated mapping-row controls to expose accessible remove semantics and expanded setup UI server tests to assert accessibility markers in the embedded HTML.
  - Changed files/tests:
    - `mama/internal/ui/static/index.html`
    - `mama/internal/ui/server_test.go`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

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
- [x] `DONE` Add signing/notarization workflow docs and automation.
  - Implemented:
    - Added a dedicated release-signing GitHub Actions workflow that signs published release assets with keyless Sigstore and uploads generated `*.sig`/`*.pem` artifacts back to each release.
    - Added optional macOS notarization automation (conditional on Apple credentials) that codesigns, notarizes, staples, and re-uploads notarized zip artifacts.
    - Added maintainer-facing signing/notarization documentation and reusable release scripts for local execution, plus release-doc checklist updates.
  - Changed files/tests:
    - `.github/workflows/release-signing.yml`
    - `scripts/release/sign-artifacts.sh`
    - `scripts/release/notarize-macos.sh`
    - `docs/SIGNING_AND_NOTARIZATION.md`
    - `docs/RELEASE_REPRODUCIBLE_BUILDS.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add optional installer/update path while preserving portable mode.
  - Implemented:
    - Added release artifact automation workflow (`release-artifacts.yml`) that builds portable archives for Linux/macOS/Windows and publishes optional update manifests and Windows installer artifacts.
    - Added Windows optional installer packaging via Inno Setup (`package-installer.ps1` + `mama-installer.iss`) with graceful skip behavior when `iscc` is unavailable so portable mode remains first-class.
    - Added reusable update-manifest generation script and updated installation/release documentation to describe portable-first distribution plus optional installer/update flows.
  - Changed files/tests:
    - `.github/workflows/release-artifacts.yml`
    - `scripts/windows/package-installer.ps1`
    - `scripts/windows/mama-installer.iss`
    - `scripts/release/generate-update-manifest.sh`
    - `docs/INSTALLATION.md`
    - `docs/RELEASE_REPRODUCIBLE_BUILDS.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Add automated changelog/release-notes generation.
  - Implemented:
    - Added a deterministic release-note generator script that builds markdown from git history between release tags, grouping conventional commit prefixes into readable sections.
    - Added a dedicated GitHub Actions workflow to auto-generate release notes on published releases, update the release body from the generated markdown, and upload a `release-notes-<tag>.md` asset.
    - Updated release integrity/reproducible-build documentation and QA checklist to include the new automated changelog/release-note flow for maintainers.
  - Changed files/tests:
    - `scripts/release/generate-release-notes.sh`
    - `.github/workflows/release-notes.yml`
    - `docs/RELEASE_REPRODUCIBLE_BUILDS.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `README.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

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
- [x] `DONE` Run v1.0 readiness review and publish acceptance criteria.
- [x] `DONE` Harden readiness/checklist docs with evidence-driven gates and automation ownership labels.
- [x] `DONE` Address readiness/release checklist reviewer follow-ups (actionable evidence + ownership clarity).
  - Status history: `IN_PROGRESS` (2026-03-02) -> `DONE` (2026-03-02).
  - Implementation plan (executed this iteration):
    1. Re-audit `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` for vague acceptance wording and missing per-item execution ownership labels.
    2. Convert checklist/readiness entries into measurable pass criteria with explicit evidence format requirements and manual owner placeholders.
    3. Run in-repo automatable release-readiness commands, capture dated command/output snippets inline, and keep non-executable manual gates pending.
    4. Recompute objective GO/NO-GO blockers from captured evidence and log shipped-vs-manual scope in the execution log.
  - Implemented:
    - Tightened `v1.0` readiness gates into per-item objective criteria with explicit execution-type labels and evidence requirements.
    - Expanded release QA checklist rows so every item is action-oriented, execution-typed, and mapped to command/evidence expectations.
    - Added fresh dated command evidence for automatable in-repo checks (tests, dependency verification, checksum script preflight/smoke) while leaving hardware/workflow/release approvals unclaimed.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Reconcile readiness/checklist docs with objective gate mapping and fresh run evidence.
- [x] `DONE` Resolve remaining readiness/checklist reviewer gaps (objective evidence blocks + section ownership matrix).
  - Status history: `IN_PROGRESS` (2026-03-02) -> `DONE` (2026-03-02).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` as the source of truth and isolate the highest-impact remaining in-repo gap: ambiguous verification/evidence wording that blocked deterministic sign-off.
    2. Convert gate/checklist wording into explicit pass/fail criteria with evidence IDs so each automatable item maps to a concrete command result and each manual item maps to a required owner artifact payload.
    3. Add explicit section-level automation ownership summary in the checklist and refresh evidence blocks from a new command run, keeping all manual/hardware/release approvals pending.
    4. Recompute readiness GO/NO-GO snapshot from refreshed evidence and document exactly what shipped versus what remains maintainer-only.
  - Implemented:
    - Added evidence-ID based acceptance criteria in `docs/V1_READINESS_REVIEW.md`, including automatable/manual completion summaries and objective blocker list tied to current run outputs.
    - Added a section ownership matrix in `docs/RELEASE_QA_CHECKLIST.md` and tightened checklist item wording so each row specifies command/reference plus mandatory evidence payload format.
    - Refreshed both documents with this run's command evidence (`go test`, `go mod verify`, checksum preflight/smoke) while preserving pending status for manual release/hardware approvals.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`
- [x] `DONE` Close remaining readiness-review doc gaps from prior PR inline comments (fresh evidence + deterministic ownership mapping).
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-audit `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` for residual non-deterministic wording still called out in prior PR inline comments (especially evidence freshness and owner mapping clarity).
    2. Tighten each automatable and manual row into explicit pass conditions with owner placeholders and required evidence payload shape.
    3. Re-run in-repo automatable readiness commands, record dated outputs inline, and update statuses based on this run only.
    4. Recompute GO/NO-GO from objective gate completion and document exactly what shipped versus what remains manual.
  - Implemented:
    - Refactored `docs/V1_READINESS_REVIEW.md` into owner-aware gate tables with refreshed dated statuses, an evidence register, and explicit objective blocker mapping for GO/NO-GO.
    - Refined `docs/RELEASE_QA_CHECKLIST.md` into stricter pass/fail wording per row, expanded section automation-boundary matrix, and added evidence-to-checklist traceability table.
    - Refreshed automatable evidence from this run (`go test`, `go mod verify`, checksum script preflight/smoke), while keeping manual/hardware/release gates pending and unclaimed.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

- [x] `DONE` Finalize readiness/checklist objective evidence contract for this iteration.
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` as the source of truth and isolate the highest-impact remaining in-repo gap: missing freshness contract + non-commanded doc-structure checks.
    2. Convert remaining vague checklist wording into executable criteria where possible, including automatable module-drift and doc-structure assertions.
    3. Re-run required automatable commands (`go test`, `go mod verify`, checksum preflight/smoke) and append fresh dated evidence blocks.
    4. Recompute readiness/checklist statuses from this run, keep manual gates pending, and log shipped-vs-manual scope.
  - Implemented:
    - Added evidence freshness rules plus command-backed doc-structure assertions in the readiness review, and refreshed all automatable gate timestamps from this run.
    - Tightened release QA checklist wording by converting dependency-drift validation into an automatable preflight check and command-anchoring documentation governance checks.
    - Captured fresh evidence for all automatable checks (`go test`, `go mod verify`, checksum preflight/smoke, module-drift, doc-anchor assertions) while preserving all manual maintainer/hardware gates as pending.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

- [x] `DONE` Address second-pass readiness/checklist PR inline comments (anchored doc assertions + fresh evidence sync).
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` and isolate the highest-impact actionable gap: doc-structure checks vulnerable to false-positive matches from embedded command snippets.
    2. Tighten automatable checklist/readiness criteria to use anchored assertions and align gate/checklist evidence mapping (`go mod verify` + module-drift preflight).
    3. Re-run automatable in-repo commands (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`) and refresh dated evidence blocks.
    4. Recompute status snapshots and record shipped-vs-manual boundary explicitly in the execution log.
  - Implemented:
    - Hardened doc-governance checks in readiness/checklist docs with anchored regex assertions so section/header checks validate document structure instead of matching quoted command text.
    - Updated readiness Gate `G3.1` criteria to require both dependency verification and module-drift preflight evidence, matching checklist `B2` + `B5` objective expectations.
    - Refreshed all automatable evidence timestamps/output snippets from this run and kept manual maintainer/hardware/workflow sign-offs pending.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

- [x] `DONE` Close latest readiness/checklist doc-review gaps (explicit section ownership labels + fresh evidence refresh).
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` and isolate the highest-impact actionable in-repo gap from reviewer feedback: section-level execution ownership was not explicitly asserted in section headings.
    2. Add explicit classification labels to checklist section headings and tighten objective criteria (`D2`/`G4.1`) so command assertions verify section-level ownership anchors, not only table columns.
    3. Re-run automatable in-repo readiness commands (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`) and refresh dated evidence/statuses inline.
    4. Recompute GO/NO-GO status from refreshed evidence and record exactly what shipped versus what remains manual in the execution log.
  - Implemented:
    - Added explicit execution-boundary labels to all release checklist section headings and updated checklist criteria so section/item ownership is verifiable via anchored commands.
    - Refreshed both readiness/checklist evidence registers with this run's command output and updated automatable gate/checklist timestamps while retaining manual maintainer gates as pending.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

- [x] `DONE` Resolve remaining readiness/checklist ownership-proof gaps (explicit owner column + fresh evidence sync).
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` and isolate the highest-impact actionable in-repo gap: owner placeholders were embedded in free-text evidence notes but not first-class checklist columns.
    2. Add explicit per-item owner fields to checklist tables and tighten readiness/checklist objective assertions so owner-column presence is command-verifiable.
    3. Re-run automatable readiness commands (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`) and refresh dated evidence/status blocks from this run.
    4. Recompute readiness snapshot and log exactly what shipped versus what remains manual maintainer sign-off.
  - Implemented:
    - Added explicit `Owner` columns across release checklist sections so each automatable/manual row has a direct accountable owner placeholder (`Codex` or `@maintainer-<name>`).
    - Tightened readiness/checklist structural assertions to require `Execution type + Owner` table anchors for governance verification.
    - Refreshed automatable evidence blocks with this run's command outputs/timestamps while keeping manual maintainer/hardware/workflow approvals pending.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`



- [x] `DONE` Refresh readiness/checklist evidence run and tighten remaining ambiguous manual criteria.
  - Status history: `IN_PROGRESS` (2026-03-03) -> `DONE` (2026-03-03).
  - Implementation plan (executed this iteration):
    1. Re-open `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md` as source of truth, then isolate the highest-impact actionable gap: remaining manual checklist rows that were owner-labeled but still too vague to verify consistently.
    2. Convert those rows into explicit pass conditions with required observable outputs/evidence payload formats while preserving manual ownership boundaries.
    3. Re-run automatable readiness commands (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift preflight, anchored `rg`) and refresh inline evidence timestamps/output snippets.
    4. Recompute objective readiness snapshot and log exactly what shipped versus what remains maintainer-only.
  - Implemented:
    - Tightened remaining manual readiness/checklist language into deterministic acceptance criteria (hardware handshake/events, interaction semantics, soak duration/cadence, artifact coverage, and release-notes completeness expectations).
    - Refreshed automatable gate/checklist evidence timestamps and status rows from this run while keeping manual maintainer/workflow/hardware gates explicitly pending.
    - Preserved `NO-GO` decision semantics by tying blockers only to unresolved manual gates and not claiming external sign-offs.
  - Changed files/tests:
    - `docs/V1_READINESS_REVIEW.md`
    - `docs/RELEASE_QA_CHECKLIST.md`
    - `docs/IMPLEMENTATION_TRACKER.md`

---

## Execution Log

- 2026-03-03: Tightened remaining ambiguous manual readiness/checklist criteria into verifiable pass conditions (explicit hardware handshake/event expectations, interaction semantics, soak duration/cadence, release artifact coverage, and release-note content requirements), refreshed automatable evidence timestamps/outputs from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift preflight, anchored `rg`), and kept manual maintainer/hardware/workflow gates pending by design.
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
- 2026-03-02: Added release signing/notarization workflow automation (`.github/workflows/release-signing.yml`), plus reusable signing/notarization scripts and maintainer docs. Updated release QA/reproducible-build docs and README references. Verified with `bash -n scripts/release/sign-artifacts.sh`, `bash -n scripts/release/notarize-macos.sh`, `cd mama && go test ./...`.

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

- 2026-03-02: Implemented `app` per-process/session backend support on Unix hosts via `pactl` sink-input controls, including selector-token runtime wiring, backend discovery updates, and contract/parser tests for app adjust/mute/list behavior; updated README/troubleshooting/support docs with current scope. Verified with `cd mama && go test ./...`.
- 2026-03-02: Implemented `group` target backend support via grouped app/session selector routing (Unix `pactl` sink-input control), including runtime selector token serialization, backend/controller contract tests, and documentation/support-matrix updates. Verified with `cd mama && go test ./...`.

- 2026-03-02: Added firmware-side encoder stress validation by extracting shared encoder/button state logic (`firmware/master/encoder_logic.h`), adding native burst/debounce stress coverage (`firmware/master/tests/encoder_logic_stress_test.cpp`), and shipping a repeatable runner script + README guidance. Verified with `scripts/firmware/run_encoder_stress_test.sh` and `cd mama && go test ./...`.
- 2026-03-02: Added I2C robustness validation for master/slave packet integrity by extracting shared slave packet accounting logic (`firmware/slave/i2c_packet.h`), adding burst-load packet integrity coverage (`firmware/slave/tests/i2c_packet_integrity_test.cpp`), and shipping a repeatable runner script (`scripts/firmware/run_i2c_robustness_test.sh`) with README guidance. Verified with `scripts/firmware/run_i2c_robustness_test.sh`, `scripts/firmware/run_encoder_stress_test.sh`, and `cd mama && go test ./...`.

- 2026-03-02: Shipped setup UI first-run guided wizard (detect board -> test port -> map knobs -> save -> verify), including step state/status wiring to existing port refresh/test/identify/save actions, added server test coverage for wizard controls in the embedded page, and updated README setup guidance. Verified with `cd mama && go test ./internal/ui ./cmd/mama-ui` and `cd mama && go test ./...`.

- 2026-03-02: Added setup UI mapping templates for streaming/conferencing/music/gaming presets with one-click apply into the existing mapping editor, extended index-page server test assertions for template controls, and documented template usage in README setup/feature sections. Verified with `cd mama && go test ./internal/ui ./cmd/mama-ui` and `cd mama && go test ./...`.

- 2026-03-02: Added optional installer/update release path while preserving portable-first distribution: shipped cross-platform release packaging workflow (`release-artifacts.yml`), optional Windows Inno Setup packaging (`package-installer.ps1` + `mama-installer.iss`), and reusable `generate-update-manifest.sh` metadata generation. Updated installation/release docs and QA checklist. Verified with `bash -n scripts/release/generate-update-manifest.sh`, `scripts/release/generate-update-manifest.sh` against a sample artifact, and `cd mama && go test ./...`.

- 2026-03-02: Added setup UI config backup/restore/import/export workflow with browser-local snapshot controls and JSON file import/export support; reused shared form serialization for save compatibility, expanded setup UI index-page assertions, and updated README usage docs. Verified with `cd mama && go test ./internal/ui ./cmd/mama-ui` and `cd mama && go test ./...`.

- 2026-03-02: Completed setup UI accessibility pass with keyboard focus-visible styling, ARIA labels + status live regions (including assertive error announcements), and accessible mapping-row remove semantics; expanded setup UI server assertions and README feature docs. Verified with `cd mama && go test ./internal/ui ./cmd/mama-ui` and `cd mama && go test ./...`.

- 2026-03-02: Added setup UI localization framework with initial English/Italian coverage (persisted language selector, localized wizard/template/status copy, and dynamic re-rendering of key UI text), expanded setup UI index-page assertions, and documented language toggle behavior in README. Verified with `cd mama && go test ./internal/ui ./cmd/mama-ui ./...`.

- 2026-03-02: Added automated changelog/release-note generation by shipping `scripts/release/generate-release-notes.sh` (deterministic tag-range markdown), plus `.github/workflows/release-notes.yml` to publish generated notes into GitHub releases and upload a markdown asset. Updated README/release docs/checklist for maintainer usage. Verified with `bash -n scripts/release/generate-release-notes.sh`, `bash -n scripts/release/generate-checksums.sh`, and manual script execution against a temporary tagged git repo fixture.

---

- 2026-03-02: Published `v1.0` readiness acceptance criteria in `docs/V1_READINESS_REVIEW.md` (including GO/NO-GO decision template and evidence expectations), linked it from the release QA checklist and README documentation index, and completed the tracker governance item. Verified with `cd mama && go test ./...`.

- 2026-03-02: Hardened `v1.0` readiness/release QA documentation into evidence-driven gates with explicit `Automatable by Codex` vs `Manual/Maintainer required` ownership labels, recorded dated outcomes for in-repo automatable checks (`cd mama && go test ./...`, `cd mama && go mod verify`), added manual owner/evidence placeholders, and tied GO/NO-GO criteria to objective blocking gates. Remaining work is explicitly manual (hardware validation, CI/release workflow runs, and maintainer approvals).

- 2026-03-02: Addressed reviewer follow-ups for readiness/release QA docs by converting remaining vague entries into objective, testable criteria; adding explicit execution-type labeling and owner/evidence requirements for every checklist section; and capturing fresh dated automatable evidence (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum-script syntax/smoke verification). GO/NO-GO remains `NO-GO` due only to manual maintainer/hardware/workflow release gates.


- 2026-03-02: Reconciled readiness/release QA docs with explicit readiness-gate mapping per checklist item, refreshed evidence-driven automatable outputs from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum syntax/smoke/verify), and standardized manual owner/evidence templates. GO/NO-GO remains `NO-GO` strictly due to pending manual maintainer/hardware/workflow gates.

- 2026-03-02: Resolved remaining readiness/checklist reviewer gaps by converting readiness/checklist rows to objective evidence IDs, adding explicit section-level automation ownership in the QA checklist, and refreshing inline command evidence from this run while keeping all manual/hardware/release approvals pending. Verified with `cd mama && go test ./...`, `cd mama && go mod verify`, and `bash -n scripts/release/generate-checksums.sh` plus smoke checksum validation.

- 2026-03-03: Closed remaining readiness/release-doc reviewer follow-ups by tightening objective pass criteria and owner/evidence payload rules in `docs/V1_READINESS_REVIEW.md` and `docs/RELEASE_QA_CHECKLIST.md`, then refreshing evidence from this run. Shipped automatable evidence updates only (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke); manual maintainer/hardware/workflow sign-offs remain pending by design.

- 2026-03-03: Finalized the remaining in-repo readiness/checklist gap by adding an explicit evidence freshness contract, converting dependency-drift and documentation-governance checks into command-verifiable rows, and refreshing automatable evidence from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module-drift preflight, and doc-anchor assertions). Manual maintainer-only gates (hardware, CI/release workflows, and final approvals) remain pending by design.

- 2026-03-03: Addressed follow-up inline doc-review comments by replacing broad doc-anchor checks with anchored structural assertions, aligning readiness Gate `G3.1` to require both `go mod verify` and module-drift preflight evidence, and refreshing automatable evidence blocks from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`). Manual maintainer-only gates remain pending by design.

- 2026-03-03: Closed latest readiness/release checklist reviewer follow-ups by adding explicit section-level execution ownership labels directly in checklist headings, tightening doc-governance anchor checks to assert those ownership markers, and refreshing all automatable evidence from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`). Shipped scope is docs/evidence hardening only; manual maintainer/hardware/workflow gates remain pending by design.


- 2026-03-03: Resolved remaining readiness/checklist ownership-proof reviewer gaps by promoting owner placeholders into first-class checklist table columns, tightening anchored doc-governance assertions to require `Execution type + Owner` headers, and refreshing automatable evidence from this run (`cd mama && go test ./...`, `cd mama && go mod verify`, checksum preflight/smoke, module drift, anchored `rg`). Shipped scope is docs/evidence hardening only; manual maintainer/hardware/workflow gates remain pending by design.


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

