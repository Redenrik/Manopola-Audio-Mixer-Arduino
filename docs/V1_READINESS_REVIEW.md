# v1.0 Readiness Review and Acceptance Criteria

This document is the release-candidate gate for publishing `v1.0.0`.

## Review Scope

A `v1.0.0` candidate is acceptable only when **all blocking gates** are complete with required evidence.

- **Automatable by Codex** gates require inline command evidence captured in this file.
- **Manual/Maintainer required** gates require owner-assigned sign-off artifacts (links + summary payload).
- **Evidence freshness rule:** only evidence captured in the latest run section may be used to mark automatable gates complete.

---

## Acceptance Gates

### Gate 1 — Product functionality baseline (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G1.1 | Automatable by Codex | `cd mama && go test ./...` completes successfully. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E1` must show only `ok` or `[no test files]` package results and no failures. |
| G1.2 | Automatable by Codex | Config compatibility behavior remains validated (`mama/internal/config` package covered). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E1` must include `ok  mama/internal/config`. |
| G1.3 | Automatable by Codex | API/runtime compatibility behavior remains validated (`mama/cmd/mama` package covered). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E1` must include `ok  mama/cmd/mama`. |
| G1.4 | Manual/Maintainer required | Representative hardware flow executes all required steps with evidence-backed outcomes: detect board, test serial port, map at least 3 knobs, save config, and verify live output changes for `master_out`. | `@maintainer-<name>` | ⬜ Pending | Dated hardware run log: board model, firmware revision, host OS, serial port, observed behavior, and `PASS`/`FAIL`. |

### Gate 2 — Reliability and resilience (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G2.1 | Automatable by Codex | Protocol compatibility handling remains covered by tests (`mama/internal/proto` package covered). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E1` must include `ok  mama/internal/proto`. |
| G2.2 | Automatable by Codex | Reconnect/backoff/metrics behavior remains covered by tests (`mama/internal/runtime` package covered). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E1` must include `ok  mama/internal/runtime`. |
| G2.3 | Manual/Maintainer required | Long-run soak completes exactly per `docs/SOAK_TEST_PLAN.md` minimum duration/cadence with artifact bundle proving zero unhandled runtime failures. | `@maintainer-<name>` | ⬜ Pending | Soak artifact bundle URL/path + run duration + explicit pass/fail summary. |

### Gate 3 — Platform and release quality (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G3.1 | Automatable by Codex | Dependency verification and module drift preflight pass (`go mod verify` exits 0 and no local `go.mod`/`go.sum` drift). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E2` must include `all modules verified`; `E4` must show explicit "no drift" confirmation output. |
| G3.2 | Automatable by Codex (artifact preflight) | Release checksum script parses and smoke-checksum flow succeeds on staged files. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E3` must include checksum generation path and `OK` verification lines. |
| G3.3 | Manual/Maintainer required | CI matrix (`.github/workflows/ci.yml`) is green for release commit on Linux/Windows/macOS. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.4 | Manual/Maintainer required | Security scan (`.github/workflows/security-scan.yml`) is green for release commit. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.5 | Manual/Maintainer required | Release assets published with checksums/signing artifacts (`*.sig`, `*.pem`) and verification output. | `@maintainer-<name>` | ⬜ Pending | Release asset links + verification command snippet + result. |
| G3.6 | Manual/Maintainer required | Optional installer/update artifacts (if shipped) validated and portable mode confirmed working. | `@maintainer-<name>` | ⬜ Pending | Platform/scenario matrix + evidence links + result. |

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G4.1 | Automatable by Codex | `docs/RELEASE_QA_CHECKLIST.md` contains section-level automation labels plus per-row `Execution type`, `Owner`, and evidence-format columns for all checklist tables. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E5` must confirm section ownership heading, section classification headings, and item execution/owner/evidence columns. |
| G4.2 | Automatable by Codex | This document defines objective gate-based GO/NO-GO criteria tied to blocking items. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E5` must confirm GO/NO-GO heading and objective blocker rule text exists. |
| G4.5 | Automatable by Codex | Every checklist data row includes explicit `Execution type` and `Owner` values (no blank accountability fields). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E6` must report `missing_rows=0` across all checklist item rows. |
| G4.6 | Automatable by Codex | Readiness/checklist evidence-run timestamps are synchronized so both docs reference the same latest execution window. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E7` must report `evidence_timestamp_sync=ok` with identical timestamp values. |
| G4.7 | Automatable by Codex | The shared readiness/checklist evidence timestamp is fresh (no older than 24 hours from this run). | Codex | ✅ Complete (2026-03-03 13:02Z) | `E8` must report `evidence_freshness_status=ok` and `age_hours<=24`. |
| G4.8 | Automatable by Codex | Checklist automatable rows mapped to Gate 4 (`D1`, `D2`, `D2a`, `D2b`, `D2c`, `D2d`) are all marked `✅ Complete` with non-empty evidence references. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E9` must report `automatable_gate4_rows=8`, `incomplete_rows=0`, and `missing_evidence_refs=0`. |
| G4.9 | Automatable by Codex | Every automatable checklist row across Sections 1-4 is complete and references an evidence ID that exists in the evidence table for this run. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E10` must report `automatable_rows=16`, `incomplete_rows=0`, `missing_evidence_ids=0`, and `unknown_evidence_ids=0`. |
| G4.10 | Automatable by Codex | Every manual checklist row uses the maintainer owner placeholder and includes structured evidence payload format guidance. | Codex | ✅ Complete (2026-03-03 13:02Z) | `E11` must report `manual_rows=19`, `bad_owner=0`, and `bad_evidence_format=0`. |
| G4.3 | Manual/Maintainer required | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | `@maintainer-<name>` | ⬜ Pending | Reviewer name + date + approval note/link. |
| G4.4 | Manual/Maintainer required | Release notes generated/reviewed and deferred non-blocking follow-ups tracked as issues. | `@maintainer-<name>` | ⬜ Pending | Release-notes artifact URL + sign-off + issue links/milestones. |

---

## Evidence Register (This Run — 2026-03-03 13:02Z)

| Evidence ID | Execution type | Command(s) | Outcome | Supports gates |
| --- | --- | --- | --- | --- |
| E1 | Automatable by Codex | `cd mama && go test ./...` | Pass | G1.1, G1.2, G1.3, G2.1, G2.2 |
| E2 | Automatable by Codex | `cd mama && go mod verify` | Pass | G3.1 |
| E3 | Automatable by Codex | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | G3.2 |
| E4 | Automatable by Codex | `git diff --exit-code -- mama/go.mod mama/go.sum` | Pass | G3.1 |
| E5 | Automatable by Codex | anchored `rg` assertions for readiness/checklist structural requirements | Pass | G4.1, G4.2 |
| E6 | Automatable by Codex | checklist accountability-field completeness assertion (`awk`) | Pass | G4.5 |
| E7 | Automatable by Codex | cross-doc evidence timestamp sync assertion (`sed`) | Pass | G4.6 |
| E8 | Automatable by Codex | cross-doc evidence freshness threshold assertion (`python3`) | Pass | G4.7 |
| E9 | Automatable by Codex | Gate-4 automatable checklist status/evidence completeness assertion (`python3`) | Pass | G4.8 |
| E10 | Automatable by Codex | full automatable-row evidence linkage assertion (`python3`) | Pass | G4.9 |
| E11 | Automatable by Codex | manual-row owner/evidence-format assertion (`python3`) | Pass | G4.10 |

### E1 — Host test suite

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	(cached)
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	(cached)
ok  	mama/internal/config	(cached)
ok  	mama/internal/proto	(cached)
ok  	mama/internal/runtime	(cached)
?   	mama/internal/serial	[no test files]
ok  	mama/internal/ui	(cached)
```

### E2 — Module dependency verification

```bash
$ cd mama && go mod verify
all modules verified
```

### E3 — Release checksum script preflight + smoke verification

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

### E5 — Documentation structure assertions

```bash
$ rg -n "^## Acceptance Gates$|^## Evidence Register \(This Run|^## Objective GO/NO-GO Decision$" docs/V1_READINESS_REVIEW.md
15:## Acceptance Gates
61:## Evidence Register (This Run — 2026-03-03 13:02Z)
239:## Objective GO/NO-GO Decision
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
80:## 5) Final Sign-off (Manual/Maintainer required)
82:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
```

### E6 — Checklist accountability-field completeness assertion

```bash
$ awk -F'|' 'BEGIN{rows=0;bad=0} /^\| [A-Z][0-9] / {rows++; et=$4; ow=$5; gsub(/^ +| +$/,"",et); gsub(/^ +| +$/,"",ow); if(et==""||ow==""){bad++; print "missing execution/owner on row: "$0}} END{print "checked_rows=" rows ", missing_rows=" bad; exit bad}' docs/RELEASE_QA_CHECKLIST.md
checked_rows=38, missing_rows=0
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
evidence_timestamp_sync=ok (2026-03-03 13:02Z)
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
if age_hours > threshold:
    raise SystemExit(f"evidence_freshness_status=fail age_hours={age_hours:.2f} threshold_hours={threshold} timestamp={ts1_raw}")
print(f"evidence_freshness_status=ok age_hours={age_hours:.2f} threshold_hours={threshold} timestamp={ts1_raw} now_utc={os.environ['NOW_UTC']}")
PY
evidence_freshness_status=ok age_hours=0.05 threshold_hours=24 timestamp=2026-03-03 13:02Z now_utc=2026-03-03 13:05Z
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
D1: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E5` must show both anchors.
D2: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E5` must show section ownership heading, section classification headings, and table header anchors.
D2a: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E6` must report `missing_rows=0`.
D2b: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E7` must report `evidence_timestamp_sync=ok (...)`.
D2c: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E8` must report `evidence_freshness_status=ok` with `age_hours<=24`.
D2d: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E9` must report `automatable_gate4_rows=8`, `incomplete_rows=0`, and `missing_evidence_refs=0`.
D2e: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E10` must report `automatable_rows=16`, `incomplete_rows=0`, `missing_evidence_ids=0`, and `unknown_evidence_ids=0`.
D2f: status=✅ Complete (2026-03-03 13:02Z); evidence=Section 6 `E11` must report `manual_rows=19`, `bad_owner=0`, and `bad_evidence_format=0`.
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
B1: status=✅ Complete (2026-03-03 13:02Z); refs=E1
B2: status=✅ Complete (2026-03-03 13:02Z); refs=E2
B5: status=✅ Complete (2026-03-03 13:02Z); refs=E4
R1: status=✅ Complete (2026-03-03 13:02Z); refs=E1
R2: status=✅ Complete (2026-03-03 13:02Z); refs=E1
A1: status=✅ Complete (2026-03-03 13:02Z); refs=E3
A2: status=✅ Complete (2026-03-03 13:02Z); refs=E3
A3: status=✅ Complete (2026-03-03 13:02Z); refs=E3
D1: status=✅ Complete (2026-03-03 13:02Z); refs=E5
D2: status=✅ Complete (2026-03-03 13:02Z); refs=E5
D2a: status=✅ Complete (2026-03-03 13:02Z); refs=E6
D2b: status=✅ Complete (2026-03-03 13:02Z); refs=E7
D2c: status=✅ Complete (2026-03-03 13:02Z); refs=E8
D2d: status=✅ Complete (2026-03-03 13:02Z); refs=E9
D2e: status=✅ Complete (2026-03-03 13:02Z); refs=E10
D2f: status=✅ Complete (2026-03-03 13:02Z); refs=E11
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
print(f"manual_rows={len(rows)} bad_owner={len(bad_owner)} bad_evidence_format={len(bad_evidence)}")
if bad_owner or bad_evidence:
    raise SystemExit(1)
PY
manual_rows=19 bad_owner=0 bad_evidence_format=0
```

### Manual evidence template (all pending manual gates)

```text
Owner: @maintainer-<name>
Date:
Gate(s):
Environment/Scope: (OS, hardware, workflow run, release tag/commit)
Evidence links:
Result: PASS | FAIL | BLOCKED
Notes:
```

---

## Objective GO/NO-GO Decision

- **GO** only if every blocking gate (`G1.x` through `G4.x`) is `✅ Complete` with required evidence attached.
- **NO-GO** if any blocking gate is `⬜ Pending`, `❌ Failed`, or `⚠️ Blocked`; each blocker requires remediation owner + target date.

## Gate Completion Snapshot

| Gate family | Automatable complete | Manual complete | Blocking pending |
| --- | --- | --- | --- |
| Gate 1 — Product functionality baseline | 3 / 3 | 0 / 1 | `G1.4` |
| Gate 2 — Reliability and resilience | 2 / 2 | 0 / 1 | `G2.3` |
| Gate 3 — Platform and release quality | 2 / 2 | 0 / 4 | `G3.3`, `G3.4`, `G3.5`, `G3.6` |
| Gate 4 — Documentation and governance readiness | 8 / 8 | 0 / 2 | `G4.3`, `G4.4` |

### Current decision snapshot (from this run)

- **Decision:** `NO-GO (expected in-repo pre-release state)`
- **Automatable gates summary:** 15/15 complete (`G1.1-G1.3`, `G2.1-G2.2`, `G3.1-G3.2`, `G4.1-G4.2`, `G4.5-G4.10`) based on `E1-E11`.
- **Manual gates summary:** 0/8 complete (`G1.4`, `G2.3`, `G3.3-G3.6`, `G4.3-G4.4`).
- **Objective blockers:** `G1.4`, `G2.3`, `G3.3`, `G3.4`, `G3.5`, `G3.6`, `G4.3`, `G4.4`.

## Sign-off Template

```text
v1.0 readiness decision: GO | NO-GO
Date:
Release candidate commit/tag:
Reviewers:
Blocking gaps (if any):
Remediation owners + target dates:
Evidence links:
```
