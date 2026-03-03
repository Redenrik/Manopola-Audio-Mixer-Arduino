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

## 1) Build & Test Baseline (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| B1 | G1.1, G1.2, G1.3, G2.1, G2.2 | Automatable by Codex | Codex | Run host test suite; pass only if all packages are `ok` or `[no test files]` and zero failing packages. | `cd mama && go test ./...` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E1` must include `mama/cmd/mama`, `mama/internal/config`, `mama/internal/proto`, `mama/internal/runtime`. |
| B2 | G3.1 | Automatable by Codex | Codex | Verify module dependency checksums; pass only if command exits 0 and prints `all modules verified`. | `cd mama && go mod verify` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E2` with exact success line. |
| B3 | G3.3 | Manual/Maintainer required | @maintainer-<name> | Confirm CI matrix workflow passed for release commit across Linux/Windows/macOS. | `.github/workflows/ci.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion. |
| B4 | G3.4 | Manual/Maintainer required | @maintainer-<name> | Confirm security scan workflow passed for release commit. | `.github/workflows/security-scan.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion. |
| B5 | G3.1 | Automatable by Codex | Codex | Confirm working tree has no unexpected local module drift before release verification. Pass only if exit code is 0. | `git diff --exit-code -- mama/go.mod mama/go.sum` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E4` with explicit "no drift" confirmation output. |

---

## 2) Runtime and Config Verification (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| R1 | G2.1 | Automatable by Codex | Codex | Confirm protocol compatibility behavior remains test-covered. | `cd mama && go test ./...` (must include `mama/internal/proto`) | ✅ Complete (2026-03-03 11:52Z) | Reuse Section 6 `E1`. |
| R2 | G1.2 | Automatable by Codex | Codex | Confirm config compatibility aliases/precedence remain test-covered. | `cd mama && go test ./...` (must include `mama/internal/config`) | ✅ Complete (2026-03-03 11:52Z) | Reuse Section 6 `E1`. |
| R3 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify setup UI round-trip works end-to-end: load page, modify at least one mapping row, save config, restart host, and confirm persisted value is reloaded. | `mama-ui` runtime smoke | ⬜ Pending | Owner `@maintainer-<name>` + OS/browser + executed steps + saved `config.yaml` snippet/path + result. |
| R4 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify serial connection against representative board/firmware with deterministic handshake evidence (`V:1` seen + `K:` events received). | Hardware smoke test | ⬜ Pending | Owner `@maintainer-<name>` + board model + firmware revision + serial port + log excerpt + result. |
| R5 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify live interaction semantics: one clockwise detent increases `master_out`, one counter-clockwise detent decreases it, and button press toggles mute state. | Hardware interaction test | ⬜ Pending | Owner `@maintainer-<name>` + timestamped video/log/screenshot + observed transitions + result. |
| R6 | G2.3 | Manual/Maintainer required | @maintainer-<name> | Execute soak validation per `docs/SOAK_TEST_PLAN.md` on release candidate build for required minimum duration and artifact cadence. | `docs/SOAK_TEST_PLAN.md` | ⬜ Pending | Owner `@maintainer-<name>` + artifact bundle path/URL + run duration + pass/fail summary. |

---

## 3) Release Artifacts and Integrity (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A1 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Validate release checksum script syntax. | `bash -n scripts/release/generate-checksums.sh` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E3` preflight command must exit without syntax errors. |
| A2 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Smoke-test checksum generation on local staged files. | `scripts/release/generate-checksums.sh <artifact_dir>` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E3` must include `wrote checksums: .../SHA256SUMS.txt`. |
| A3 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Verify generated checksum manifest against staged files. | `(cd <artifact_dir> && sha256sum -c SHA256SUMS.txt)` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E3` must show `OK` for each staged file. |
| A4 | G3.5 | Manual/Maintainer required | @maintainer-<name> | Confirm intended platform artifacts were published for every declared target OS/arch in release plan. | GitHub Release assets | ⬜ Pending | Owner `@maintainer-<name>` + release URL + platform coverage note. |
| A5 | G3.5 | Manual/Maintainer required | @maintainer-<name> | Confirm signing artifacts (`*.sig`, `*.pem`) are attached and valid. | Signing workflow / `scripts/release/sign-artifacts.sh` | ⬜ Pending | Owner `@maintainer-<name>` + workflow/log URL + asset links + verification output + result. |
| A6 | G3.5 | Manual/Maintainer required | @maintainer-<name> | If macOS app artifacts are shipped, confirm notarized/stapled zips are attached. | Notarization workflow / `scripts/release/notarize-macos.sh` | ⬜ Pending | Owner `@maintainer-<name>` + notarization ticket/log + asset links + result. |
| A7 | G3.6 | Manual/Maintainer required | @maintainer-<name> | Confirm portable mode works (binary + adjacent `config.yaml`, no installer required). | Portable smoke test | ⬜ Pending | Owner `@maintainer-<name>` + executed commands + runtime result summary. |
| A8 | G3.6 | Manual/Maintainer required | @maintainer-<name> | If installer artifacts are shipped, validate install/uninstall and portable fallback. | Installer matrix test | ⬜ Pending | Owner `@maintainer-<name>` + matrix results table + logs + result. |
| A9 | G3.6 | Manual/Maintainer required | @maintainer-<name> | If update manifest is shipped, validate each entry URL resolves and checksum/size fields exactly match published assets. | `*-update-manifest.json` verification | ⬜ Pending | Owner `@maintainer-<name>` + verification output + release links + result. |

---

## 4) Documentation and Governance (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| D1 | G4.2 | Automatable by Codex | Codex | Confirm readiness doc defines objective gates and GO/NO-GO criteria. Pass only if `rg` finds gate and GO/NO-GO section anchors. | `rg -n "^## Acceptance Gates$|^## Objective GO/NO-GO Decision$|^## Evidence Register" docs/V1_READINESS_REVIEW.md` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E5` must show both anchors. |
| D2 | G4.1 | Automatable by Codex | Codex | Confirm this checklist labels each section/item with execution type and evidence payload requirements. Pass only if section ownership heading, all section classification headings, and `Execution type` + `Owner` table columns are present. | `rg -n "^## Section ownership and automation boundary$|^## [1-5]\) .*\((Mixed|Manual/Maintainer required)\)$|^\| ID \| Readiness gate mapping \| Execution type \| Owner \|" docs/RELEASE_QA_CHECKLIST.md` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E5` must show section ownership heading, section classification headings, and table header anchors. |
| D2a | G4.5 | Automatable by Codex | Codex | Confirm every checklist item row has explicit accountability fields populated. Pass only if data-row scan reports zero missing `Execution type`/`Owner` cells. | `awk -F'|' 'BEGIN{rows=0;bad=0} /^\| [A-Z][0-9] / {rows++; et=$4; ow=$5; gsub(/^ +| +$/,"",et); gsub(/^ +| +$/,"",ow); if(et==""||ow==""){bad++; print "missing execution/owner on row: "$0}} END{print "checked_rows=" rows ", missing_rows=" bad; exit bad}' docs/RELEASE_QA_CHECKLIST.md` | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E6` must report `missing_rows=0`. |
| D2b | G4.6 | Automatable by Codex | Codex | Confirm readiness/checklist evidence timestamps match the same run window. Pass only if extracted timestamps are identical and non-empty. | `sed` timestamp extraction + equality assertion across readiness/checklist docs | ✅ Complete (2026-03-03 11:52Z) | Section 6 `E7` must report `evidence_timestamp_sync=ok (...)`. |
| D3 | G4.4 | Manual/Maintainer required | @maintainer-<name> | Generate/review release notes and validate they include breaking-change callouts, upgrade steps, and linked issue references when applicable. | `scripts/release/generate-release-notes.sh` + release PR review | ⬜ Pending | Owner `@maintainer-<name>` + generated notes artifact URL + reviewer sign-off. |
| D4 | G4.3 | Manual/Maintainer required | @maintainer-<name> | Record support/deprecation review against `docs/SUPPORT_POLICY.md`. | Support review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link. |
| D5 | G4.3 | Manual/Maintainer required | @maintainer-<name> | Record security-impacting changes review against `SECURITY.md`. | Security review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link. |
| D6 | G4.4 | Manual/Maintainer required | @maintainer-<name> | Capture open non-blocking follow-ups as issues with milestones. | GitHub issues | ⬜ Pending | Owner `@maintainer-<name>` + issue links + milestone labels. |
| D7 | G4.x summary | Manual/Maintainer required | @maintainer-<name> | Record `docs/V1_READINESS_REVIEW.md` GO/NO-GO decision for candidate commit/tag in release PR. | Release PR sign-off block | ⬜ Pending | Owner `@maintainer-<name>` + completed sign-off block + commit SHA/tag. |

---

## 5) Final Sign-off (Manual/Maintainer required)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| S1 | Final release approval | Manual/Maintainer required | @maintainer-<name> | Final release candidate approved by maintainers. | ⬜ Pending | Owner `@maintainer-<name>` + explicit approval comment links. |
| S2 | Final release publication | Manual/Maintainer required | @maintainer-<name> | Tag created and release published with checksums attached. | ⬜ Pending | Owner `@maintainer-<name>` + tag/release URL + checksum asset link. |

---

## 6) Evidence Captured This Run (2026-03-03 11:52Z)

| Evidence ID | Command(s) | Outcome | Used by checklist IDs |
| --- | --- | --- | --- |
| E1 | `cd mama && go test ./...` | Pass | B1, R1, R2 |
| E2 | `cd mama && go mod verify` | Pass | B2 |
| E3 | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | A1, A2, A3 |
| E4 | `git diff --exit-code -- mama/go.mod mama/go.sum` | Pass | B5 |
| E5 | anchored `rg` assertions for readiness/checklist structural requirements | Pass | D1, D2 |
| E6 | checklist accountability-field completeness assertion (`awk`) | Pass | D2a |
| E7 | cross-doc evidence timestamp sync assertion (`sed`) | Pass | D2b |

### E1 — Host test suite

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	0.025s
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	0.021s
ok  	mama/internal/config	0.049s
ok  	mama/internal/proto	0.016s
ok  	mama/internal/runtime	0.016s
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

### E4 — Module drift preflight

```bash
$ git diff --exit-code -- mama/go.mod mama/go.sum && echo "no go module drift in working tree"
no go module drift in working tree
```

### E5 — Readiness/checklist structure anchor checks

```bash
$ rg -n "^## Acceptance Gates$|^## Objective GO/NO-GO Decision$|^## Evidence Register" docs/V1_READINESS_REVIEW.md
15:## Acceptance Gates
58:## Evidence Register (This Run — 2026-03-03 11:52Z)
168:## Objective GO/NO-GO Decision
$ rg -n "^## Section ownership and automation boundary$|^## [1-5]\) .*\((Mixed|Manual/Maintainer required)\)$|^\| ID \| Readiness gate mapping \| Execution type \| Owner \|" docs/RELEASE_QA_CHECKLIST.md
9:## Section ownership and automation boundary
21:## 1) Build & Test Baseline (Mixed)
23:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
33:## 2) Runtime and Config Verification (Mixed)
35:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
46:## 3) Release Artifacts and Integrity (Mixed)
48:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
62:## 4) Documentation and Governance (Mixed)
64:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
78:## 5) Final Sign-off (Manual/Maintainer required)
80:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
```

### E6 — Checklist accountability-field completeness assertion

```bash
$ awk -F'|' 'BEGIN{rows=0;bad=0} /^\| [A-Z][0-9] / {rows++; et=$4; ow=$5; gsub(/^ +| +$/,"",et); gsub(/^ +| +$/,"",ow); if(et==""||ow==""){bad++; print "missing execution/owner on row: "$0}} END{print "checked_rows=" rows ", missing_rows=" bad; exit bad}' docs/RELEASE_QA_CHECKLIST.md
checked_rows=36, missing_rows=0
```

### E7 — Cross-document evidence timestamp synchronization assertion

```bash
$ ts1=$(sed -n 's/^## Evidence Register (This Run — \(.*\))$/\1/p' docs/V1_READINESS_REVIEW.md)
$ ts2=$(sed -n 's/^## 6) Evidence Captured This Run (\(.*\))$/\1/p' docs/RELEASE_QA_CHECKLIST.md)
$ if [ "$ts1" = "$ts2" ] && [ -n "$ts1" ]; then
>   echo "evidence_timestamp_sync=ok ($ts1)"
> else
>   echo "evidence_timestamp_sync=fail (readiness='$ts1' checklist='$ts2')"
>   exit 1
> fi
evidence_timestamp_sync=ok (2026-03-03 11:52Z)
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
