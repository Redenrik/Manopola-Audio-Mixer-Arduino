# Roadmap to a Fully Developed and Finished MAMA

This roadmap is a gap-driven plan based on the current repository state (firmware + Go daemon + setup UI + docs).

## 1) Definition of "Finished"

A finished MAMA release should provide:
- complete feature parity for all declared mapping targets (`master_out`, `mic_in`, `line_in`, `app`, `group`)
- reliable operation across supported OSes and common hardware variations
- a polished non-technical user experience (install, setup, run, recover)
- release engineering maturity (CI, signed artifacts, rollback-safe updates)
- maintainable code with clear ownership and long-term testing strategy

---

## 2) Current Major Gaps

### 2.1 Feature completeness gaps
- audio backend currently supports `master_out` only; other targets are placeholders
- no in-app discovery/selection workflow for per-app targets and groups
- no profile system (use-case presets, multi-layout setups)
- no firmware/host protocol version negotiation

### 2.2 Reliability and quality gaps
- no hardware-in-the-loop automated test suite
- limited integration tests for serial edge cases and long-running behavior
- no resilience features (auto reconnect backoff strategy visibility, health telemetry)

### 2.3 Packaging and operations gaps
- Windows-first packaging exists, but no parity scripts/pipelines for Linux/macOS
- no code signing / notarization process documented or automated
- no installer + updater path for mainstream users

### 2.4 Product and UX gaps
- setup UI is functional but not yet guided/onboarding-first
- no backup/restore/import/export of configuration in UI
- no localization framework implementation yet
- no accessibility checklist or formal UX acceptance criteria

### 2.5 Security and governance gaps
- no explicit threat model or security policy document
- no dependency/vulnerability scanning pipeline
- no clear support lifecycle/versioning policy

---

## 3) Delivery Roadmap

## Phase A — Close Core Functional Gaps (v0.4)
**Goal:** make all declared targets truly usable.

### Scope
1. Implement `mic_in` and `line_in` controls in platform backends.
2. Implement `app` target (per-process/session volume) with process matching rules.
3. Implement `group` target with explicit schema for grouping multiple app/process selectors.
4. Extend `/api/targets` to return dynamic, selectable targets from host OS.
5. Add validation and conflict rules for app/group mappings (duplicate/overlap handling).

### Exit criteria
- every `TargetType` accepted by config is executable at runtime
- setup UI can list, pick, and persist all target types without manual YAML edits
- target-specific tests added for parser/config/backend behavior

## Phase B — Reliability, Recovery, and Protocol Hardening (v0.5)
**Goal:** ensure predictable behavior under real-world instability.

### Scope
1. Add protocol versioning metadata and compatibility checks.
2. Add serial fixture replay integration tests (bursts, malformed lines, disconnect/reconnect).
3. Add firmware stress tests for high-rate encoder activity and button bounce edge cases.
4. Add structured diagnostics: connection state, dropped-line counters, backend action errors.
5. Implement robust auto-reconnect with bounded retry/backoff + clear user feedback.

### Exit criteria
- long-run soak tests complete without state corruption
- recovery from USB reconnect validated and documented
- protocol changes governed by compatibility tests

## Phase C — UX and Setup Maturity (v0.6)
**Goal:** make setup effortless for non-technical users.

### Scope
1. Guided onboarding wizard (detect board -> test port -> map knobs -> save -> run check).
2. Mapping templates (streaming, conferencing, music production, gaming).
3. Config backup/restore and import/export in setup UI.
4. Localization support implementation (start with EN/IT as documented intention).
5. Accessibility improvements (keyboard navigation, labels, contrast, status messaging).

### Exit criteria
- first-time setup completed without terminal or file editing
- UX smoke tests for onboarding and identify mode pass on fresh machines
- config recovery flow tested end-to-end

## Phase D — Release Engineering and Distribution (v0.7)
**Goal:** production-grade distribution and repeatable releases.

### Scope
1. CI matrix builds for Windows/Linux/macOS with artifact publishing.
2. Reproducible builds + checksums + signed release metadata.
3. Windows signing and macOS notarization workflows.
4. Optional installer path (keeping portable mode as first-class).
5. Automated release notes/changelog generation and semver tagging policy.

### Exit criteria
- one-command or one-workflow release process from tagged commit
- downloadable signed artifacts for each supported OS
- documented rollback and upgrade instructions

## Phase E — Sustainability and Community Readiness (v1.0)
**Goal:** transition from project to maintainable product.

### Scope
1. Define support policy (OS versions, hardware revisions, deprecation window).
2. Add SECURITY policy, vulnerability reporting process, and dependency scanning.
3. Expand contributor docs with architecture decision records (ADRs).
4. Add issue templates, bug triage labels, and release QA checklist.
5. Create telemetry-lite optional diagnostics bundle for troubleshooting (opt-in, privacy-safe).

### Exit criteria
- stable v1.0 release candidate passes functional, reliability, and packaging gates
- maintainership workflow documented and sustainable
- security and support lifecycle explicitly published

---

## 4) Cross-Cutting Workstreams (run in parallel)

### Firmware workstream
- configurable debounce/step behavior per encoder model
- optional calibration mode for direction/noise correction
- watchdog/recovery strategy for I2C communication issues

### Desktop daemon workstream
- backend abstraction split by capabilities (system, input, app-session)
- richer error taxonomy and user-safe messaging
- deterministic logging format and optional log rotation

### UI workstream
- componentize current single-page static UI for maintainability
- inline validation with actionable hints
- stateful connection monitor and visual runtime diagnostics

### QA workstream
- test matrix by OS + common audio stack variants
- fixture-driven serial parser fuzzing
- regression suite for mapping behaviors and mute toggles

---

## 5) Prioritization: What to Build First

### Highest impact (do first)
1. Implement missing audio targets (`mic_in`, `line_in`, `app`, `group`).
2. Add reconnect/reliability instrumentation and protocol hardening.
3. Add CI matrix with automated tests and release artifacts.

### Medium impact
4. Improve onboarding UX + templates + backup/restore.
5. Add signing/notarization and installer/update options.

### Long-term polish
6. Localization depth, accessibility audits, governance and community workflows.

---

## 6) Suggested Milestone Checklist

- [ ] M1: all target types implemented and selectable in UI
- [ ] M2: protocol versioning + serial integration suite + soak tests
- [ ] M3: guided setup + template mappings + backup/restore
- [ ] M4: multi-OS signed release pipeline + checksums + release notes automation
- [ ] M5: security/support policies + v1.0 readiness review

When all five milestones are complete, MAMA can be considered "fully developed" for a v1.0 product baseline.
