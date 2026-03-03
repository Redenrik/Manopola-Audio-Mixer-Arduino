# Release QA Checklist

Use this checklist before tagging or publishing a release artifact.

> Status legend: `✅ Complete`, `⬜ Pending`, `❌ Failed`, `⚠️ Blocked`

---

## Section ownership and automation boundary

| Section | Classification | Automatable by Codex scope | Manual/Maintainer required scope |
| --- | --- | --- | --- |
| 1) Build & Test Baseline | Mixed | Run local test/dependency commands and capture outputs. | Review CI runs and release PR approval context. |
| 2) Runtime and Config Verification | Mixed | Validate in-repo protocol/config/runtime package coverage through tests. | Execute hardware interaction and soak validation. |
| 3) Release Artifacts and Integrity | Mixed | Run checksum script preflight and local smoke verification. | Validate published release artifacts/signatures/notarization/installers. |
| 4) Documentation and Governance | Mixed | Verify doc structure/objective readiness rules are present. | Complete maintainer reviews/sign-offs and issue tracking. |
| 5) Final Sign-off | Manual/Maintainer required | _None_ | Final approval + publication decision. |

---

## 1) Build & Test Baseline

| ID | Readiness gate mapping | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| B1 | G1.1, G1.2, G1.3, G2.1, G2.2 | Automatable by Codex | Run host test suite; pass only if all packages are `ok` or `[no test files]` and zero failing packages. | `cd mama && go test ./...` | ✅ Complete (2026-03-03) | Section 6 `E1` must include `mama/cmd/mama`, `mama/internal/config`, `mama/internal/proto`, `mama/internal/runtime`. |
| B2 | G3.1 | Automatable by Codex | Verify module dependency checksums; pass only if command exits 0 and prints `all modules verified`. | `cd mama && go mod verify` | ✅ Complete (2026-03-03) | Section 6 `E2` with exact success line. |
| B3 | G3.3 | Manual/Maintainer required | Confirm CI matrix workflow passed for release commit across Linux/Windows/macOS. | `.github/workflows/ci.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion. |
| B4 | G3.4 | Manual/Maintainer required | Confirm security scan workflow passed for release commit. | `.github/workflows/security-scan.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion. |
| B5 | G3.1 | Manual/Maintainer required | Confirm no unreviewed `go.mod`/`go.sum` drift remains in release PR. | Release PR diff review | ⬜ Pending | Owner `@maintainer-<name>` + explicit review note that dependency diff is acceptable/empty. |

---

## 2) Runtime and Config Verification

| ID | Readiness gate mapping | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| R1 | G2.1 | Automatable by Codex | Confirm protocol compatibility behavior remains test-covered. | `cd mama && go test ./...` (must include `mama/internal/proto`) | ✅ Complete (2026-03-03) | Reuse Section 6 `E1`. |
| R2 | G1.2 | Automatable by Codex | Confirm config compatibility aliases/precedence remain test-covered. | `cd mama && go test ./...` (must include `mama/internal/config`) | ✅ Complete (2026-03-03) | Reuse Section 6 `E1`. |
| R3 | G1.4 | Manual/Maintainer required | Verify setup UI loads and saves a valid config in representative environment. | `mama-ui` runtime smoke | ⬜ Pending | Owner `@maintainer-<name>` + OS/browser + executed steps + saved `config.yaml` snippet/path + result. |
| R4 | G1.4 | Manual/Maintainer required | Verify serial connection against representative board/firmware. | Hardware smoke test | ⬜ Pending | Owner `@maintainer-<name>` + board model + firmware revision + serial port + log excerpt + result. |
| R5 | G1.4 | Manual/Maintainer required | Verify knob rotation changes `master_out` and button toggles mute in live run. | Hardware interaction test | ⬜ Pending | Owner `@maintainer-<name>` + timestamped video/log/screenshot + observed transitions + result. |
| R6 | G2.3 | Manual/Maintainer required | Execute soak validation per `docs/SOAK_TEST_PLAN.md` on release candidate build. | `docs/SOAK_TEST_PLAN.md` | ⬜ Pending | Owner `@maintainer-<name>` + artifact bundle path/URL + run duration + pass/fail summary. |

---

## 3) Release Artifacts and Integrity

| ID | Readiness gate mapping | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| A1 | G3.2 | Automatable by Codex (artifact preflight) | Validate release checksum script syntax. | `bash -n scripts/release/generate-checksums.sh` | ✅ Complete (2026-03-03) | Section 6 `E3` preflight command must exit without syntax errors. |
| A2 | G3.2 | Automatable by Codex (artifact preflight) | Smoke-test checksum generation on local staged files. | `scripts/release/generate-checksums.sh <artifact_dir>` | ✅ Complete (2026-03-03) | Section 6 `E3` must include `wrote checksums: .../SHA256SUMS.txt`. |
| A3 | G3.2 | Automatable by Codex (artifact preflight) | Verify generated checksum manifest against staged files. | `(cd <artifact_dir> && sha256sum -c SHA256SUMS.txt)` | ✅ Complete (2026-03-03) | Section 6 `E3` must show `OK` for each staged file. |
| A4 | G3.5 | Manual/Maintainer required | Confirm intended platform artifacts were published. | GitHub Release assets | ⬜ Pending | Owner `@maintainer-<name>` + release URL + platform coverage note. |
| A5 | G3.5 | Manual/Maintainer required | Confirm signing artifacts (`*.sig`, `*.pem`) are attached and valid. | Signing workflow / `scripts/release/sign-artifacts.sh` | ⬜ Pending | Owner `@maintainer-<name>` + workflow/log URL + asset links + verification output + result. |
| A6 | G3.5 | Manual/Maintainer required | If macOS app artifacts are shipped, confirm notarized/stapled zips are attached. | Notarization workflow / `scripts/release/notarize-macos.sh` | ⬜ Pending | Owner `@maintainer-<name>` + notarization ticket/log + asset links + result. |
| A7 | G3.6 | Manual/Maintainer required | Confirm portable mode works (binary + adjacent `config.yaml`, no installer required). | Portable smoke test | ⬜ Pending | Owner `@maintainer-<name>` + executed commands + runtime result summary. |
| A8 | G3.6 | Manual/Maintainer required | If installer artifacts are shipped, validate install/uninstall and portable fallback. | Installer matrix test | ⬜ Pending | Owner `@maintainer-<name>` + matrix results table + logs + result. |
| A9 | G3.6 | Manual/Maintainer required | If update manifest is shipped, validate URL/checksum/size fields against published assets. | `*-update-manifest.json` verification | ⬜ Pending | Owner `@maintainer-<name>` + verification output + release links + result. |

---

## 4) Documentation and Governance

| ID | Readiness gate mapping | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| D1 | G4.2 | Automatable by Codex | Confirm readiness doc defines objective gates and GO/NO-GO criteria. | `docs/V1_READINESS_REVIEW.md` | ✅ Complete (2026-03-03) | Readiness doc includes objective gate tables + GO/NO-GO blocker rule. |
| D2 | G4.1 | Automatable by Codex | Confirm this checklist labels each section/item with execution type and evidence payload requirements. | `docs/RELEASE_QA_CHECKLIST.md` | ✅ Complete (2026-03-03) | Section ownership matrix and per-item execution/evidence columns present. |
| D3 | G4.4 | Manual/Maintainer required | Generate/review release notes and validate upgrade guidance completeness. | `scripts/release/generate-release-notes.sh` + release PR review | ⬜ Pending | Owner `@maintainer-<name>` + generated notes artifact URL + reviewer sign-off. |
| D4 | G4.3 | Manual/Maintainer required | Record support/deprecation review against `docs/SUPPORT_POLICY.md`. | Support review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link. |
| D5 | G4.3 | Manual/Maintainer required | Record security-impacting changes review against `SECURITY.md`. | Security review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link. |
| D6 | G4.4 | Manual/Maintainer required | Capture open non-blocking follow-ups as issues with milestones. | GitHub issues | ⬜ Pending | Owner `@maintainer-<name>` + issue links + milestone labels. |
| D7 | G4.x summary | Manual/Maintainer required | Record `docs/V1_READINESS_REVIEW.md` GO/NO-GO decision for candidate commit/tag in release PR. | Release PR sign-off block | ⬜ Pending | Owner `@maintainer-<name>` + completed sign-off block + commit SHA/tag. |

---

## 5) Final Sign-off (Manual/Maintainer required)

| ID | Readiness gate mapping | Execution type | Actionable verification step | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- |
| S1 | Final release approval | Manual/Maintainer required | Final release candidate approved by maintainers. | ⬜ Pending | Owner `@maintainer-<name>` + explicit approval comment links. |
| S2 | Final release publication | Manual/Maintainer required | Tag created and release published with checksums attached. | ⬜ Pending | Owner `@maintainer-<name>` + tag/release URL + checksum asset link. |

---

## 6) Evidence Captured This Run (2026-03-03)

| Evidence ID | Command(s) | Outcome | Used by checklist IDs |
| --- | --- | --- | --- |
| E1 | `cd mama && go test ./...` | Pass | B1, R1, R2 |
| E2 | `cd mama && go mod verify` | Pass | B2 |
| E3 | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | A1, A2, A3 |

### E1 — Host test suite

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	0.021s
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	0.020s
ok  	mama/internal/config	0.059s
ok  	mama/internal/proto	0.018s
ok  	mama/internal/runtime	0.019s
?   	mama/internal/serial	[no test files]
ok  	mama/internal/ui	0.026s
```

### E2 — Dependency verification

```bash
$ cd mama && go mod verify
all modules verified
```

### E3 — Checksum script preflight + smoke verification

```bash
$ bash -n scripts/release/generate-checksums.sh
$ rm -rf /tmp/mama-release-smoke && mkdir -p /tmp/mama-release-smoke
$ printf 'alpha\n' > /tmp/mama-release-smoke/a.txt
$ printf 'beta\n' > /tmp/mama-release-smoke/b.txt
$ scripts/release/generate-checksums.sh /tmp/mama-release-smoke
wrote checksums: /tmp/mama-release-smoke/SHA256SUMS.txt
$ (cd /tmp/mama-release-smoke && sha256sum -c SHA256SUMS.txt)
a.txt: OK
b.txt: OK
```

### Manual evidence template (for all pending manual rows)

```text
Owner: @maintainer-<name>
Date:
Checklist IDs:
Environment/Scope:
Evidence links:
Result: PASS | FAIL | BLOCKED
Notes:
```
