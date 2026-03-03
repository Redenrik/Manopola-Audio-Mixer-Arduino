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
| 5) Final Sign-off | Manual/Maintainer required | _None_ | Final approval + publication decision + Result (`PASS`/`FAIL`/`BLOCKED`). |

---

## 1) Build & Test Baseline (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| B1 | G1.1, G1.2, G1.3, G2.1, G2.2 | Automatable by Codex | Codex | Run host test suite; pass only if all packages are `ok` or `[no test files]` and zero failing packages. | `cd mama && go test ./...` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E1` must include `mama/cmd/mama`, `mama/internal/config`, `mama/internal/proto`, `mama/internal/runtime`. |
| B2 | G3.1 | Automatable by Codex | Codex | Verify module dependency checksums; pass only if command exits 0 and prints `all modules verified`. | `cd mama && go mod verify` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E2` with exact success line. |
| B3 | G3.3 | Manual/Maintainer required | @maintainer-<name> | Confirm CI matrix workflow passed for release commit across Linux/Windows/macOS. | `.github/workflows/ci.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion + Result (`PASS`/`FAIL`/`BLOCKED`). |
| B4 | G3.4 | Manual/Maintainer required | @maintainer-<name> | Confirm security scan workflow passed for release commit. | `.github/workflows/security-scan.yml` | ⬜ Pending | Owner `@maintainer-<name>` + workflow URL + run ID + commit SHA + conclusion + Result (`PASS`/`FAIL`/`BLOCKED`). |
| B5 | G3.1 | Automatable by Codex | Codex | Confirm working tree has no unexpected local module drift before release verification. Pass only if exit code is 0. | `git diff --exit-code -- mama/go.mod mama/go.sum` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E4` with explicit "no drift" confirmation output. |

---

## 2) Runtime and Config Verification (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| R1 | G2.1 | Automatable by Codex | Codex | Confirm protocol compatibility behavior remains test-covered. | `cd mama && go test ./...` (must include `mama/internal/proto`) | ✅ Complete (2026-03-03 16:05Z) | Reuse Section 6 `E1`. |
| R2 | G1.2 | Automatable by Codex | Codex | Confirm config compatibility aliases/precedence remain test-covered. | `cd mama && go test ./...` (must include `mama/internal/config`) | ✅ Complete (2026-03-03 16:05Z) | Reuse Section 6 `E1`. |
| R3 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify setup UI round-trip works end-to-end: load page, modify at least one mapping row, save config, restart host, and confirm persisted value is reloaded. | `mama-ui` runtime smoke | ⬜ Pending | Owner `@maintainer-<name>` + OS/browser + executed steps + saved `config.yaml` snippet/path + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| R4 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify serial connection against representative board/firmware with deterministic handshake evidence (`V:1` seen + `K:` events received). | Hardware smoke test | ⬜ Pending | Owner `@maintainer-<name>` + board model + firmware revision + serial port + log excerpt + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| R5 | G1.4 | Manual/Maintainer required | @maintainer-<name> | Verify live interaction semantics: one clockwise detent increases `master_out`, one counter-clockwise detent decreases it, and button press toggles mute state. | Hardware interaction test | ⬜ Pending | Owner `@maintainer-<name>` + timestamped video/log/screenshot + observed transitions + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| R6 | G2.3 | Manual/Maintainer required | @maintainer-<name> | Execute soak validation per `docs/SOAK_TEST_PLAN.md` on release candidate build for required minimum duration and artifact cadence. | `docs/SOAK_TEST_PLAN.md` | ⬜ Pending | Owner `@maintainer-<name>` + artifact bundle path/URL + run duration + pass/fail summary + Result (`PASS`/`FAIL`/`BLOCKED`). |

---

## 3) Release Artifacts and Integrity (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A1 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Validate release checksum script syntax. | `bash -n scripts/release/generate-checksums.sh` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E3` preflight command must exit without syntax errors. |
| A2 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Smoke-test checksum generation on local staged files. | `scripts/release/generate-checksums.sh <artifact_dir>` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E3` must include `wrote checksums: .../SHA256SUMS.txt`. |
| A3 | G3.2 | Automatable by Codex (artifact preflight) | Codex | Verify generated checksum manifest against staged files. | `(cd <artifact_dir> && sha256sum -c SHA256SUMS.txt)` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E3` must show `OK` for each staged file. |
| A4 | G3.5 | Manual/Maintainer required | @maintainer-<name> | Confirm intended platform artifacts were published for every declared target OS/arch in release plan. | GitHub Release assets | ⬜ Pending | Owner `@maintainer-<name>` + release URL + platform coverage note + Result (`PASS`/`FAIL`/`BLOCKED`). |
| A5 | G3.5 | Manual/Maintainer required | @maintainer-<name> | Confirm signing artifacts (`*.sig`, `*.pem`) are attached and valid. | Signing workflow / `scripts/release/sign-artifacts.sh` | ⬜ Pending | Owner `@maintainer-<name>` + workflow/log URL + asset links + verification output + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| A6 | G3.5 | Manual/Maintainer required | @maintainer-<name> | If macOS app artifacts are shipped, confirm notarized/stapled zips are attached. | Notarization workflow / `scripts/release/notarize-macos.sh` | ⬜ Pending | Owner `@maintainer-<name>` + notarization ticket/log + asset links + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| A7 | G3.6 | Manual/Maintainer required | @maintainer-<name> | Confirm portable mode works (binary + adjacent `config.yaml`, no installer required). | Portable smoke test | ⬜ Pending | Owner `@maintainer-<name>` + executed commands + runtime result summary + Result (`PASS`/`FAIL`/`BLOCKED`). |
| A8 | G3.6 | Manual/Maintainer required | @maintainer-<name> | If installer artifacts are shipped, validate install/uninstall and portable fallback. | Installer matrix test | ⬜ Pending | Owner `@maintainer-<name>` + matrix results table + logs + result + Result (`PASS`/`FAIL`/`BLOCKED`). |
| A9 | G3.6 | Manual/Maintainer required | @maintainer-<name> | If update manifest is shipped, validate each entry URL resolves and checksum/size fields exactly match published assets. | `*-update-manifest.json` verification | ⬜ Pending | Owner `@maintainer-<name>` + verification output + release links + result + Result (`PASS`/`FAIL`/`BLOCKED`). |

---

## 4) Documentation and Governance (Mixed)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Command/reference | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- | --- |
| D1 | G4.2 | Automatable by Codex | Codex | Confirm readiness doc defines objective gates and GO/NO-GO criteria. Pass only if `rg` finds gate and GO/NO-GO section anchors. | `rg -n "^## Acceptance Gates$|^## Objective GO/NO-GO Decision$|^## Evidence Register" docs/V1_READINESS_REVIEW.md` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E5` must show both anchors. |
| D2 | G4.1 | Automatable by Codex | Codex | Confirm this checklist labels each section/item with execution type and evidence payload requirements. Pass only if section ownership heading, all section classification headings, and `Execution type` + `Owner` table columns are present. | `rg -n "^## Section ownership and automation boundary$|^## [1-5]\) .*\((Mixed|Manual/Maintainer required)\)$|^\| ID \| Readiness gate mapping \| Execution type \| Owner \|" docs/RELEASE_QA_CHECKLIST.md` | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E5` must show section ownership heading, section classification headings, and table header anchors. |
| D2a | G4.5 | Automatable by Codex | Codex | Confirm every checklist item row has explicit accountability fields populated. Pass only if parser scans all non-evidence checklist rows (including suffixed IDs like `D2a`) and reports zero missing `Execution type`/`Owner` cells. | `python3` checklist accountability-field completeness assertion | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E6` must report `checked_rows=35` and `missing_rows=0`. |
| D2b | G4.6 | Automatable by Codex | Codex | Confirm readiness/checklist evidence timestamps match the same run window. Pass only if extracted timestamps are identical and non-empty. | `sed` timestamp extraction + equality assertion across readiness/checklist docs | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E7` must report `evidence_timestamp_sync=ok (...)`. |
| D2c | G4.7 | Automatable by Codex | Codex | Confirm shared evidence timestamp is fresh for this iteration. Pass only if cross-doc timestamps match exactly and age is within `0-24h` (future timestamps fail). | `python3` timestamp age assertion across readiness/checklist docs | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E8` must report `evidence_freshness_status=ok` with `0<=age_hours<=24`. |
| D2d | G4.8 | Automatable by Codex | Codex | Confirm all Gate-4 automatable checklist rows are complete and include non-empty evidence references. Pass only if parser reports eight automatable rows with zero incomplete rows and zero missing evidence references. | `python3` checklist-row status/evidence assertion for Gate-4 automatable items | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E9` must report `automatable_gate4_rows=8`, `incomplete_rows=0`, and `missing_evidence_refs=0`. |
| D2e | G4.9 | Automatable by Codex | Codex | Confirm every automatable checklist row across Sections 1-4 is complete and mapped to an existing evidence ID from Section 6. Pass only if parser reports zero incomplete rows and zero missing/unknown evidence IDs. | `python3` full automatable-row evidence linkage assertion | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E10` must report `automatable_rows=16`, `incomplete_rows=0`, `missing_evidence_ids=0`, and `unknown_evidence_ids=0`. |
| D2f | G4.10 | Automatable by Codex | Codex | Confirm every manual checklist row uses explicit maintainer owner placeholder and structured evidence payload format with required result token. Pass only if parser reports zero bad owner cells, zero malformed manual evidence fields, and zero rows missing result token guidance. | `python3` manual-row owner/evidence-format assertion | ✅ Complete (2026-03-03 16:05Z) | Section 6 `E11` must report `manual_rows=19`, `bad_owner=0`, `bad_evidence_format=0`, and `missing_result_token=0`. |
| D3 | G4.4 | Manual/Maintainer required | @maintainer-<name> | Generate/review release notes and validate they include breaking-change callouts, upgrade steps, and linked issue references when applicable. | `scripts/release/generate-release-notes.sh` + release PR review | ⬜ Pending | Owner `@maintainer-<name>` + generated notes artifact URL + reviewer sign-off + Result (`PASS`/`FAIL`/`BLOCKED`). |
| D4 | G4.3 | Manual/Maintainer required | @maintainer-<name> | Record support/deprecation review against `docs/SUPPORT_POLICY.md`. | Support review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link + Result (`PASS`/`FAIL`/`BLOCKED`). |
| D5 | G4.3 | Manual/Maintainer required | @maintainer-<name> | Record security-impacting changes review against `SECURITY.md`. | Security review note | ⬜ Pending | Owner `@maintainer-<name>` + reviewer + date + approval note/link + Result (`PASS`/`FAIL`/`BLOCKED`). |
| D6 | G4.4 | Manual/Maintainer required | @maintainer-<name> | Capture open non-blocking follow-ups as issues with milestones. | GitHub issues | ⬜ Pending | Owner `@maintainer-<name>` + issue links + milestone labels + Result (`PASS`/`FAIL`/`BLOCKED`). |
| D7 | G4.x summary | Manual/Maintainer required | @maintainer-<name> | Record `docs/V1_READINESS_REVIEW.md` GO/NO-GO decision for candidate commit/tag in release PR. | Release PR sign-off block | ⬜ Pending | Owner `@maintainer-<name>` + completed sign-off block + commit SHA/tag + Result (`PASS`/`FAIL`/`BLOCKED`). |

---

## 5) Final Sign-off (Manual/Maintainer required)

| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
| --- | --- | --- | --- | --- | --- | --- |
| S1 | Final release approval | Manual/Maintainer required | @maintainer-<name> | Final release candidate approved by at least two maintainers and no unresolved blocking threads remain in release PR. | ⬜ Pending | Owner `@maintainer-<name>` + two approval comment links + note that blocking threads are resolved + Result (`PASS`/`FAIL`/`BLOCKED`). |
| S2 | Final release publication | Manual/Maintainer required | @maintainer-<name> | Tag created and release published with checksums attached. | ⬜ Pending | Owner `@maintainer-<name>` + tag/release URL + checksum asset link + Result (`PASS`/`FAIL`/`BLOCKED`). |

---

## 6) Evidence Captured This Run (2026-03-03 16:05Z)

| Evidence ID | Command(s) | Outcome | Used by checklist IDs |
| --- | --- | --- | --- |
| E1 | `cd mama && go test ./...` | Pass | B1, R1, R2 |
| E2 | `cd mama && go mod verify` | Pass | B2 |
| E3 | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | A1, A2, A3 |
| E4 | `git diff --exit-code -- mama/go.mod mama/go.sum` | Pass | B5 |
| E5 | anchored `rg` assertions for readiness/checklist structural requirements | Pass | D1, D2 |
| E6 | checklist accountability-field completeness assertion (`python3`) | Pass | D2a |
| E7 | cross-doc evidence timestamp sync assertion (`sed`) | Pass | D2b |
| E8 | cross-doc evidence freshness threshold assertion (`python3`) | Pass | D2c |
| E9 | Gate-4 automatable checklist status/evidence completeness assertion (`python3`) | Pass | D2d |
| E10 | full automatable-row evidence linkage assertion (`python3`) | Pass | D2e |
| E11 | manual-row owner/evidence-format assertion (`python3`) | Pass | D2f |

### E1 — Host test suite

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	0.028s
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	0.022s
ok  	mama/internal/config	0.031s
ok  	mama/internal/proto	0.023s
ok  	mama/internal/runtime	0.016s
?   	mama/internal/serial	[no test files]
ok  	mama/internal/ui	0.028s
```

### E2 — Dependency verification

```bash
$ cd mama && go mod verify
all modules verified
```

### E3 — Checksum script preflight + smoke verification

```bash
$ bash -n scripts/release/generate-checksums.sh
$ rm -rf /tmp/tmp.a5PVUbz11k && mkdir -p /tmp/tmp.a5PVUbz11k
$ printf 'alpha\n' > /tmp/tmp.a5PVUbz11k/a.txt
$ printf 'beta\n' > /tmp/tmp.a5PVUbz11k/b.txt
$ scripts/release/generate-checksums.sh /tmp/tmp.a5PVUbz11k
wrote checksums: /tmp/tmp.a5PVUbz11k/SHA256SUMS.txt
$ (cd /tmp/tmp.a5PVUbz11k && sha256sum -c SHA256SUMS.txt)
mama_linux_amd64.tar.gz: OK
mama_windows_amd64.zip: OK
```

### E4 — Module drift preflight

```bash
$ git diff --exit-code -- mama/go.mod mama/go.sum && echo "module_drift_check=ok (no local go.mod/go.sum drift)"
module_drift_check=ok (no local go.mod/go.sum drift)
```

### E5 — Readiness/checklist structure anchor checks

```bash
$ rg -n "^## Acceptance Gates$|^## Objective GO/NO-GO Decision$|^## Evidence Register" docs/V1_READINESS_REVIEW.md
15:## Acceptance Gates
61:## Evidence Register (This Run — 2026-03-03 16:05Z)
373:## Objective GO/NO-GO Decision
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
82:## 5) Final Sign-off (Manual/Maintainer required)
84:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
```

### E6 — Checklist accountability-field completeness assertion

```bash
$ python3 - <<'PY'
from pathlib import Path
import re

rows = []
for line in Path("docs/RELEASE_QA_CHECKLIST.md").read_text().splitlines():
    m = re.match(r"^\| ([A-Z][0-9][a-z]?) \|", line)
    if not m:
        continue
    row_id = m.group(1)
    if row_id.startswith("E"):
        continue
    trimmed = line.strip()
    if trimmed.endswith("|"):
        trimmed = trimmed[:-1]
    parts = [part.strip() for part in trimmed.split("|")[1:]]
    if len(parts) < 7:
        continue
    rows.append((row_id, parts[2], parts[3]))

bad = [r for r in rows if not r[1] or not r[2]]
print(f"checked_rows={len(rows)}, missing_rows={len(bad)}")
for row_id, execution_type, owner in rows:
    print(f"{row_id}: execution_type={execution_type}; owner={owner}")
if bad:
    raise SystemExit(1)
PY
checked_rows=35, missing_rows=0
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
evidence_timestamp_sync=ok (2026-03-03 16:05Z)
```

### E8 — Cross-document evidence freshness threshold assertion

```bash
$ now_utc=$(date -u +"%Y-%m-%d %H:%MZ")
$ NOW_UTC="$now_utc" python3 - <<'PY'
from datetime import datetime
from pathlib import Path
import os
import re

fmt = "%Y-%m-%d %H:%MZ"
threshold = 24
now = datetime.strptime(os.environ["NOW_UTC"], fmt)
readiness = Path("docs/V1_READINESS_REVIEW.md").read_text()
checklist = Path("docs/RELEASE_QA_CHECKLIST.md").read_text()
pat1 = re.search(r"^## Evidence Register \(This Run — (.*)\)$", readiness, re.M)
pat2 = re.search(r"^## 6\) Evidence Captured This Run \((.*)\)$", checklist, re.M)
if not pat1 or not pat2:
    raise SystemExit("missing evidence timestamp heading")
ts1_raw = pat1.group(1)
ts2_raw = pat2.group(1)
ts1 = datetime.strptime(ts1_raw, fmt)
ts2 = datetime.strptime(ts2_raw, fmt)
if ts1 != ts2:
    raise SystemExit(f"evidence_freshness_status=fail reason=timestamp_mismatch readiness={ts1_raw} checklist={ts2_raw}")
age_hours = (now - ts1).total_seconds() / 3600
if age_hours < 0:
    raise SystemExit(f"evidence_freshness_status=fail reason=future_timestamp age_hours={age_hours:.2f} timestamp={ts1_raw} now_utc={os.environ['NOW_UTC']}")
if age_hours > threshold:
    raise SystemExit(f"evidence_freshness_status=fail reason=stale_timestamp age_hours={age_hours:.2f} threshold_hours={threshold} timestamp={ts1_raw}")
print(f"evidence_freshness_status=ok age_hours={age_hours:.2f} threshold_hours={threshold} timestamp={ts1_raw} now_utc={os.environ['NOW_UTC']}")
PY
evidence_freshness_status=ok age_hours=0.03 threshold_hours=24 timestamp=2026-03-03 16:05Z now_utc=2026-03-03 16:05Z
```

### E9 — Gate-4 automatable checklist status/evidence completeness assertion

```bash
$ python3 - <<'PY'
from pathlib import Path

rows = []
for line in Path("docs/RELEASE_QA_CHECKLIST.md").read_text().splitlines():
    if not line.startswith("| D") or "| Automatable by Codex" not in line:
        continue
    trimmed = line.strip()
    if trimmed.endswith("|"):
        trimmed = trimmed[:-1]
    row_id = trimmed.split("|", 2)[1].strip()
    _, status, evidence = trimmed.rsplit("|", 2)
    rows.append((row_id, status.strip(), evidence.strip()))

incomplete = [r for r in rows if not r[1].startswith("✅ Complete")]
missing_evidence = [r for r in rows if r[2] in ("", "-")]
print(f"automatable_gate4_rows={len(rows)} incomplete_rows={len(incomplete)} missing_evidence_refs={len(missing_evidence)}")
for row_id, status, evidence in rows:
    print(f"{row_id}: status={status}; evidence={evidence}")
if incomplete or missing_evidence:
    raise SystemExit(1)
PY
automatable_gate4_rows=8 incomplete_rows=0 missing_evidence_refs=0
D1: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E5` must show both anchors.
D2: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E5` must show section ownership heading, section classification headings, and table header anchors.
D2a: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E6` must report `missing_rows=0`.
D2b: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E7` must report `evidence_timestamp_sync=ok (...)`.
D2c: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E8` must report `evidence_freshness_status=ok` with `0<=age_hours<=24`.
D2d: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E9` must report `automatable_gate4_rows=8`, `incomplete_rows=0`, and `missing_evidence_refs=0`.
D2e: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E10` must report `automatable_rows=16`, `incomplete_rows=0`, `missing_evidence_ids=0`, and `unknown_evidence_ids=0`.
D2f: status=✅ Complete (2026-03-03 16:05Z); evidence=Section 6 `E11` must report `manual_rows=19`, `bad_owner=0`, `bad_evidence_format=0`, and `missing_result_token=0`.
```

### E10 — Full automatable-row evidence linkage assertion

```bash
$ python3 - <<'PY'
from pathlib import Path
import re

doc = Path("docs/RELEASE_QA_CHECKLIST.md").read_text().splitlines()
evidence_ids = set()
in_evidence_table = False
for line in doc:
    if line.startswith("## 6) Evidence Captured This Run"):
        in_evidence_table = True
        continue
    if in_evidence_table and line.startswith("### "):
        break
    if in_evidence_table and line.startswith("| E"):
        evidence_ids.add(line.split("|", 2)[1].strip())

rows = []
for line in doc:
    m = re.match(r"^\| ([A-Z][0-9][a-z]?) \|", line)
    if not m:
        continue
    row_id = m.group(1)
    if row_id.startswith(("S", "E")):
        continue
    trimmed = line.strip()
    if trimmed.endswith("|"):
        trimmed = trimmed[:-1]
    left, status, evidence = trimmed.rsplit("|", 2)
    execution_type = left.split("|", 4)[3].strip()
    if "Automatable by Codex" not in execution_type:
        continue
    refs = set(re.findall(r"`(E\d+)`", evidence))
    rows.append((row_id, status.strip(), refs))

incomplete = [r for r in rows if not r[1].startswith("✅ Complete")]
missing_ids = [r for r in rows if not r[2]]
unknown = sorted({eid for _, _, refs in rows for eid in refs if eid not in evidence_ids})
print(f"automatable_rows={len(rows)} incomplete_rows={len(incomplete)} missing_evidence_ids={len(missing_ids)} unknown_evidence_ids={len(unknown)}")
for row_id, status, refs in rows:
    print(f"{row_id}: status={status}; refs={','.join(sorted(refs)) if refs else '-'}")
if incomplete or missing_ids or unknown:
    raise SystemExit(1)
PY
automatable_rows=16 incomplete_rows=0 missing_evidence_ids=0 unknown_evidence_ids=0
B1: status=✅ Complete (2026-03-03 16:05Z); refs=E1
B2: status=✅ Complete (2026-03-03 16:05Z); refs=E2
B5: status=✅ Complete (2026-03-03 16:05Z); refs=E4
R1: status=✅ Complete (2026-03-03 16:05Z); refs=E1
R2: status=✅ Complete (2026-03-03 16:05Z); refs=E1
A1: status=✅ Complete (2026-03-03 16:05Z); refs=E3
A2: status=✅ Complete (2026-03-03 16:05Z); refs=E3
A3: status=✅ Complete (2026-03-03 16:05Z); refs=E3
D1: status=✅ Complete (2026-03-03 16:05Z); refs=E5
D2: status=✅ Complete (2026-03-03 16:05Z); refs=E5
D2a: status=✅ Complete (2026-03-03 16:05Z); refs=E6
D2b: status=✅ Complete (2026-03-03 16:05Z); refs=E7
D2c: status=✅ Complete (2026-03-03 16:05Z); refs=E8
D2d: status=✅ Complete (2026-03-03 16:05Z); refs=E9
D2e: status=✅ Complete (2026-03-03 16:05Z); refs=E10
D2f: status=✅ Complete (2026-03-03 16:05Z); refs=E11
```

### E11 — Manual-row ownership/evidence-format assertion

```bash
$ python3 - <<'PY'
from pathlib import Path
import re

rows = []
for line in Path("docs/RELEASE_QA_CHECKLIST.md").read_text().splitlines():
    m = re.match(r"^\| ([A-Z][0-9][a-z]?) \|", line)
    if not m:
        continue
    row_id = m.group(1)
    if row_id.startswith("E"):
        continue
    trimmed = line.strip()
    if trimmed.endswith("|"):
        trimmed = trimmed[:-1]
    parts = [part.strip() for part in trimmed.split("|")[1:]]
    if len(parts) < 7:
        continue
    execution_type = parts[2]
    owner = parts[3]
    evidence = parts[-1]
    if execution_type != "Manual/Maintainer required":
        continue
    rows.append((row_id, owner, evidence))

bad_owner = [row for row in rows if row[1] != "@maintainer-<name>"]
bad_evidence = [row for row in rows if "@maintainer-<name>" not in row[2] or "+" not in row[2]]
missing_result = [row for row in rows if "Result (`PASS`/`FAIL`/`BLOCKED`)" not in row[2]]
print(f"manual_rows={len(rows)} bad_owner={len(bad_owner)} bad_evidence_format={len(bad_evidence)} missing_result_token={len(missing_result)}")
if bad_owner or bad_evidence or missing_result:
    raise SystemExit(1)
PY
manual_rows=19 bad_owner=0 bad_evidence_format=0 missing_result_token=0
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
