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
| G1.1 | Automatable by Codex | `cd mama && go test ./...` completes successfully. | Codex | ✅ Complete (2026-03-03 11:38Z) | `E1` must show only `ok` or `[no test files]` package results and no failures. |
| G1.2 | Automatable by Codex | Config compatibility behavior remains validated (`mama/internal/config` package covered). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E1` must include `ok  mama/internal/config`. |
| G1.3 | Automatable by Codex | API/runtime compatibility behavior remains validated (`mama/cmd/mama` package covered). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E1` must include `ok  mama/cmd/mama`. |
| G1.4 | Manual/Maintainer required | Representative hardware flow executes all required steps with evidence-backed outcomes: detect board, test serial port, map at least 3 knobs, save config, and verify live output changes for `master_out`. | `@maintainer-<name>` | ⬜ Pending | Dated hardware run log: board model, firmware revision, host OS, serial port, observed behavior, and `PASS`/`FAIL`. |

### Gate 2 — Reliability and resilience (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G2.1 | Automatable by Codex | Protocol compatibility handling remains covered by tests (`mama/internal/proto` package covered). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E1` must include `ok  mama/internal/proto`. |
| G2.2 | Automatable by Codex | Reconnect/backoff/metrics behavior remains covered by tests (`mama/internal/runtime` package covered). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E1` must include `ok  mama/internal/runtime`. |
| G2.3 | Manual/Maintainer required | Long-run soak completes exactly per `docs/SOAK_TEST_PLAN.md` minimum duration/cadence with artifact bundle proving zero unhandled runtime failures. | `@maintainer-<name>` | ⬜ Pending | Soak artifact bundle URL/path + run duration + explicit pass/fail summary. |

### Gate 3 — Platform and release quality (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G3.1 | Automatable by Codex | Dependency verification and module drift preflight pass (`go mod verify` exits 0 and no local `go.mod`/`go.sum` drift). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E2` must include `all modules verified`; `E4` must show explicit "no drift" confirmation output. |
| G3.2 | Automatable by Codex (artifact preflight) | Release checksum script parses and smoke-checksum flow succeeds on staged files. | Codex | ✅ Complete (2026-03-03 11:38Z) | `E3` must include checksum generation path and `OK` verification lines. |
| G3.3 | Manual/Maintainer required | CI matrix (`.github/workflows/ci.yml`) is green for release commit on Linux/Windows/macOS. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.4 | Manual/Maintainer required | Security scan (`.github/workflows/security-scan.yml`) is green for release commit. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.5 | Manual/Maintainer required | Release assets published with checksums/signing artifacts (`*.sig`, `*.pem`) and verification output. | `@maintainer-<name>` | ⬜ Pending | Release asset links + verification command snippet + result. |
| G3.6 | Manual/Maintainer required | Optional installer/update artifacts (if shipped) validated and portable mode confirmed working. | `@maintainer-<name>` | ⬜ Pending | Platform/scenario matrix + evidence links + result. |

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G4.1 | Automatable by Codex | `docs/RELEASE_QA_CHECKLIST.md` contains section-level automation labels plus per-row `Execution type`, `Owner`, and evidence-format columns for all checklist tables. | Codex | ✅ Complete (2026-03-03 11:38Z) | `E5` must confirm section ownership heading, section classification headings, and item execution/owner/evidence columns. |
| G4.2 | Automatable by Codex | This document defines objective gate-based GO/NO-GO criteria tied to blocking items. | Codex | ✅ Complete (2026-03-03 11:38Z) | `E5` must confirm GO/NO-GO heading and objective blocker rule text exists. |
| G4.5 | Automatable by Codex | Every checklist data row includes explicit `Execution type` and `Owner` values (no blank accountability fields). | Codex | ✅ Complete (2026-03-03 11:38Z) | `E6` must report `missing_rows=0` across all checklist item rows. |
| G4.3 | Manual/Maintainer required | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | `@maintainer-<name>` | ⬜ Pending | Reviewer name + date + approval note/link. |
| G4.4 | Manual/Maintainer required | Release notes generated/reviewed and deferred non-blocking follow-ups tracked as issues. | `@maintainer-<name>` | ⬜ Pending | Release-notes artifact URL + sign-off + issue links/milestones. |

---

## Evidence Register (This Run — 2026-03-03 11:38Z)

| Evidence ID | Execution type | Command(s) | Outcome | Supports gates |
| --- | --- | --- | --- | --- |
| E1 | Automatable by Codex | `cd mama && go test ./...` | Pass | G1.1, G1.2, G1.3, G2.1, G2.2 |
| E2 | Automatable by Codex | `cd mama && go mod verify` | Pass | G3.1 |
| E3 | Automatable by Codex | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | G3.2 |
| E4 | Automatable by Codex | `git diff --exit-code -- mama/go.mod mama/go.sum` | Pass | G3.1 |
| E5 | Automatable by Codex | anchored `rg` assertions for readiness/checklist structural requirements | Pass | G4.1, G4.2 |
| E6 | Automatable by Codex | checklist accountability-field completeness assertion (`awk`) | Pass | G4.5 |

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
56:## Evidence Register (This Run — 2026-03-03 11:38Z)
143:## Objective GO/NO-GO Decision
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
76:## 5) Final Sign-off (Manual/Maintainer required)
78:| ID | Readiness gate mapping | Execution type | Owner | Actionable verification step | Current status | Evidence / required format |
```

### E6 — Checklist accountability-field completeness assertion

```bash
$ awk -F'|' 'BEGIN{rows=0;bad=0} /^\| [A-Z][0-9] / {rows++; et=$4; ow=$5; gsub(/^ +| +$/,"",et); gsub(/^ +| +$/,"",ow); if(et==""||ow==""){bad++; print "missing execution/owner on row: "$0}} END{print "checked_rows=" rows ", missing_rows=" bad; exit bad}' docs/RELEASE_QA_CHECKLIST.md
checked_rows=35, missing_rows=0
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
| Gate 4 — Documentation and governance readiness | 3 / 3 | 0 / 2 | `G4.3`, `G4.4` |

### Current decision snapshot (from this run)

- **Decision:** `NO-GO (expected in-repo pre-release state)`
- **Automatable gates summary:** 10/10 complete (`G1.1-G1.3`, `G2.1-G2.2`, `G3.1-G3.2`, `G4.1-G4.2`, `G4.5`) based on `E1-E6`.
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
