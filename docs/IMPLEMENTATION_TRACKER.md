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
- [ ] `TODO` Implement `mic_in` backend support (Windows + Unix strategy decision documented).
- [ ] `TODO` Implement `line_in` backend support.
- [ ] `TODO` Implement `app` per-process/session volume control.
- [ ] `TODO` Implement `group` target for grouped app/session controls.
- [ ] `TODO` Extend `/api/targets` to expose real discoverable targets (not placeholders).

### B. Config and mapping model maturity
- [ ] `TODO` Define and validate robust schema for `app/group` selectors (exact match, wildcard, executable name, etc.).
- [ ] `TODO` Add conflict/precedence rules for overlapping mappings.
- [ ] `TODO` Add profile support (multiple mapping sets with active profile selection).

---

## 2) Reliability / Protocol / Runtime Hardening

- [x] `DONE` Add protocol versioning and compatibility checks between firmware and host.
  - Implemented:
    - Added `V:<version>` protocol hello event support in firmware and host parser.
    - Added host compatibility gating to drop control events when firmware announces an unsupported protocol version, with clear logs.
    - Added parser/compatibility tests and updated protocol docs.
- [ ] `TODO` Implement reconnect strategy with bounded backoff and clear state transitions.
- [ ] `TODO` Add runtime metrics/counters (parse errors, dropped events, reconnect count, backend failures).
- [ ] `TODO` Add deterministic structured logs for troubleshooting.
- [ ] `TODO` Add long-run soak test plan and scripted verification artifacts.

---

## 3) Test Coverage Expansion

### A. Host-side tests
- [ ] `TODO` Add integration tests with recorded serial fixtures (valid bursts + malformed lines + disconnect cases).
- [ ] `TODO` Add backend contract tests for each supported target type.
- [ ] `TODO` Add config migration/compatibility tests for future schema evolution.

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

- [ ] `TODO` Add CI matrix builds (Windows/Linux/macOS) with test execution.
- [ ] `TODO` Add release artifact checksums and reproducible-build notes.
- [ ] `TODO` Add signing/notarization workflow docs and automation.
- [ ] `TODO` Add optional installer/update path while preserving portable mode.
- [ ] `TODO` Add automated changelog/release-notes generation.

---

## 6) Security, Governance, and v1.0 Readiness

- [ ] `TODO` Add `SECURITY.md` with disclosure workflow.
- [ ] `TODO` Add dependency and vulnerability scanning in CI.
- [ ] `TODO` Define support matrix and deprecation/versioning policy.
- [ ] `TODO` Add issue/PR templates and release QA checklist.
- [ ] `TODO` Run v1.0 readiness review and publish acceptance criteria.

---

## Execution Log

- 2026-03-01: Tracker created from roadmap gaps; all items initialized as `TODO`.
- 2026-03-01: Implemented protocol versioning handshake (`V:1`) and host compatibility checks. Updated firmware boot output, host parser/runtime gating, setup UI event labeling, and parser tests. Verified with `go test ./...`.

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
