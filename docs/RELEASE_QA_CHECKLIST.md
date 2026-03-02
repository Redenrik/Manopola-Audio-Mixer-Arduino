# Release QA Checklist

Use this checklist before tagging or publishing a release artifact.

> Status legend: `✅ Complete`, `⬜ Pending`, `❌ Failed`, `⚠️ Blocked`

---

## 1) Build & Test Baseline

| ID | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- |
| B1 | Automatable by Codex | Run full host test suite from module root and confirm all test packages pass. | `cd mama && go test ./...` | ✅ Complete (2026-03-02) | Output block in Section 6 must include `mama/cmd/mama`, `mama/internal/config`, `mama/internal/proto`, `mama/internal/runtime`. |
| B2 | Automatable by Codex | Verify module dependency checksums and module cache integrity. | `cd mama && go mod verify` | ✅ Complete (2026-03-02) | Output block in Section 6 must include `all modules verified`. |
| B3 | Manual/Maintainer required | Confirm CI matrix workflow passed for release commit. | `.github/workflows/ci.yml` | ⬜ Pending | Owner: `@maintainer-<name>`; provide workflow URL + run ID + commit SHA. |
| B4 | Manual/Maintainer required | Confirm security scan workflow passed for release commit. | `.github/workflows/security-scan.yml` | ⬜ Pending | Owner: `@maintainer-<name>`; provide workflow URL + run ID + commit SHA. |
| B5 | Manual/Maintainer required | Confirm no unreviewed `go.mod`/`go.sum` drift remains in release PR. | Release PR diff review | ⬜ Pending | Owner: `@maintainer-<name>`; explicit review comment acknowledging dependency diff status. |

---

## 2) Runtime and Config Verification

| ID | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- |
| R1 | Automatable by Codex | Confirm protocol compatibility behavior is still test-covered (accept `V:1`, reject mismatches). | `cd mama && go test ./...` (includes `mama/internal/proto`) | ✅ Complete (2026-03-02) | Reuse Section 6 `go test` output block. |
| R2 | Automatable by Codex | Confirm config compatibility aliases and precedence behavior are still test-covered. | `cd mama && go test ./...` (includes `mama/internal/config`) | ✅ Complete (2026-03-02) | Reuse Section 6 `go test` output block. |
| R3 | Manual/Maintainer required | Verify setup UI loads and saves a valid config in representative environment. | `mama-ui` runtime smoke | ⬜ Pending | Owner: `@maintainer-<name>`; include OS/browser, executed steps, and saved `config.yaml` snippet/path. |
| R4 | Manual/Maintainer required | Verify serial connection against representative board/firmware. | Hardware smoke test | ⬜ Pending | Owner: `@maintainer-<name>`; include board model, firmware revision, serial port, and log excerpt. |
| R5 | Manual/Maintainer required | Verify knob rotation changes `master_out` and button toggles mute in live run. | Hardware interaction test | ⬜ Pending | Owner: `@maintainer-<name>`; include timestamped video/log/screenshot evidence. |
| R6 | Manual/Maintainer required | Execute soak validation per `docs/SOAK_TEST_PLAN.md` on release candidate build. | `docs/SOAK_TEST_PLAN.md` | ⬜ Pending | Owner: `@maintainer-<name>`; include artifact bundle path/URL and pass/fail summary. |

---

## 3) Release Artifacts and Integrity

| ID | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- |
| A1 | Automatable by Codex (artifact preflight) | Validate release checksum script syntax. | `bash -n scripts/release/generate-checksums.sh` | ✅ Complete (2026-03-02) | Output in Section 6 (no syntax errors). |
| A2 | Automatable by Codex (artifact preflight) | Smoke-test checksum generation on local staged files. | `scripts/release/generate-checksums.sh <artifact_dir>` | ✅ Complete (2026-03-02) | Section 6 output showing `wrote checksums: .../SHA256SUMS.txt`. |
| A3 | Automatable by Codex (artifact preflight) | Verify generated checksum manifest against files. | `(cd <artifact_dir> && sha256sum -c SHA256SUMS.txt)` | ✅ Complete (2026-03-02) | Section 6 output showing `OK` for each file. |
| A4 | Manual/Maintainer required | Confirm intended platform artifacts were published. | GitHub Release assets | ⬜ Pending | Owner: `@maintainer-<name>`; release URL + platform coverage note. |
| A5 | Manual/Maintainer required | Confirm signing artifacts (`*.sig`, `*.pem`) are attached and valid for release assets. | Signing workflow / `scripts/release/sign-artifacts.sh` | ⬜ Pending | Owner: `@maintainer-<name>`; workflow/log URL + asset links + verification snippet. |
| A6 | Manual/Maintainer required | If macOS app artifacts are shipped, confirm notarized/stapled zips are attached. | Notarization workflow / `scripts/release/notarize-macos.sh` | ⬜ Pending | Owner: `@maintainer-<name>`; notarization ticket/log + asset links. |
| A7 | Manual/Maintainer required | Confirm portable mode works (binary + adjacent `config.yaml`, no installer required). | Portable smoke test | ⬜ Pending | Owner: `@maintainer-<name>`; executed commands + runtime result summary. |
| A8 | Manual/Maintainer required | If installer artifacts are shipped, validate install/uninstall and portable fallback. | Installer matrix test | ⬜ Pending | Owner: `@maintainer-<name>`; matrix results table + logs. |
| A9 | Manual/Maintainer required | If update manifest is shipped, validate URL/checksum/size fields against published assets. | `*-update-manifest.json` verification | ⬜ Pending | Owner: `@maintainer-<name>`; verification output + release links. |

---

## 4) Documentation and Governance

| ID | Execution type | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- |
| D1 | Automatable by Codex | Confirm readiness doc defines objective gates and GO/NO-GO decision criteria. | `docs/V1_READINESS_REVIEW.md` | ✅ Complete (2026-03-02) | This checklist iteration references updated gate-based review doc. |
| D2 | Automatable by Codex | Confirm this checklist labels each item with execution type and evidence requirements. | `docs/RELEASE_QA_CHECKLIST.md` | ✅ Complete (2026-03-02) | This checklist iteration includes execution type/evidence columns. |
| D3 | Manual/Maintainer required | Generate/review release notes and validate upgrade guidance completeness. | `scripts/release/generate-release-notes.sh` + release PR review | ⬜ Pending | Owner: `@maintainer-<name>`; generated notes artifact URL + reviewer sign-off. |
| D4 | Manual/Maintainer required | Record support/deprecation review against `docs/SUPPORT_POLICY.md`. | Support review note | ⬜ Pending | Owner: `@maintainer-<name>`; reviewer + date + approval note/link. |
| D5 | Manual/Maintainer required | Record security-impacting changes review against `SECURITY.md`. | Security review note | ⬜ Pending | Owner: `@maintainer-<name>`; reviewer + date + approval note/link. |
| D6 | Manual/Maintainer required | Capture open non-blocking follow-ups as issues with milestones. | GitHub issues | ⬜ Pending | Owner: `@maintainer-<name>`; issue links + milestone labels. |
| D7 | Manual/Maintainer required | Record `docs/V1_READINESS_REVIEW.md` GO/NO-GO decision for candidate commit/tag in release PR. | Release PR sign-off block | ⬜ Pending | Owner: `@maintainer-<name>`; completed sign-off block + commit SHA/tag. |

---

## 5) Final Sign-off (Manual/Maintainer required)

| ID | Execution type | Actionable verification step | Current status | Evidence / required format |
| --- | --- | --- | --- | --- |
| S1 | Manual/Maintainer required | Final release candidate approved by maintainers. | ⬜ Pending | Owner: `@maintainer-<name>`; explicit approval comment links. |
| S2 | Manual/Maintainer required | Tag created and release published with checksums attached. | ⬜ Pending | Owner: `@maintainer-<name>`; tag/release URL + checksum asset link. |

---

## 6) Evidence Captured This Run (2026-03-02)

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	0.012s
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	0.015s
ok  	mama/internal/config	0.028s
ok  	mama/internal/proto	0.011s
ok  	mama/internal/runtime	0.011s
?   	mama/internal/serial	[no test files]
ok  	mama/internal/ui	0.019s

$ cd mama && go mod verify
all modules verified

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
