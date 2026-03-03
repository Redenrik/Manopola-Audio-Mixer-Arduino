# v1.0 Readiness Review and Acceptance Criteria

This document is the release-candidate gate for publishing `v1.0.0`.

## Review Scope

A `v1.0.0` candidate is acceptable only when **all blocking gates** are complete with required evidence.

- **Automatable by Codex** gates require inline command evidence captured in this file.
- **Manual/Maintainer required** gates require owner-assigned sign-off artifacts (links + summary payload).

---

## Acceptance Gates

### Gate 1 — Product functionality baseline (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G1.1 | Automatable by Codex | `cd mama && go test ./...` completes successfully. | Codex | ✅ Complete (2026-03-03) | `E1` must show only `ok` or `[no test files]` package results and no failures. |
| G1.2 | Automatable by Codex | Config compatibility behavior remains validated (`mama/internal/config` package covered). | Codex | ✅ Complete (2026-03-03) | `E1` must include `ok  mama/internal/config`. |
| G1.3 | Automatable by Codex | API/runtime compatibility behavior remains validated (`mama/cmd/mama` package covered). | Codex | ✅ Complete (2026-03-03) | `E1` must include `ok  mama/cmd/mama`. |
| G1.4 | Manual/Maintainer required | Representative hardware flow succeeds: detect board → test port → map knobs → save → verify output changes. | `@maintainer-<name>` | ⬜ Pending | Dated hardware run log: board model, firmware revision, host OS, serial port, observed behavior, and `PASS`/`FAIL`. |

### Gate 2 — Reliability and resilience (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G2.1 | Automatable by Codex | Protocol compatibility handling remains covered by tests (`mama/internal/proto` package covered). | Codex | ✅ Complete (2026-03-03) | `E1` must include `ok  mama/internal/proto`. |
| G2.2 | Automatable by Codex | Reconnect/backoff/metrics behavior remains covered by tests (`mama/internal/runtime` package covered). | Codex | ✅ Complete (2026-03-03) | `E1` must include `ok  mama/internal/runtime`. |
| G2.3 | Manual/Maintainer required | Long-run soak for release candidate completed according to `docs/SOAK_TEST_PLAN.md`. | `@maintainer-<name>` | ⬜ Pending | Soak artifact bundle URL/path + run duration + explicit pass/fail summary. |

### Gate 3 — Platform and release quality (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G3.1 | Automatable by Codex | Dependency verification passes (`go mod verify` exits 0). | Codex | ✅ Complete (2026-03-03) | `E2` must include `all modules verified`. |
| G3.2 | Automatable by Codex (artifact preflight) | Release checksum script parses and smoke-checksum flow succeeds on staged files. | Codex | ✅ Complete (2026-03-03) | `E3` must include checksum generation path and `OK` verification lines. |
| G3.3 | Manual/Maintainer required | CI matrix (`.github/workflows/ci.yml`) is green for release commit on Linux/Windows/macOS. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.4 | Manual/Maintainer required | Security scan (`.github/workflows/security-scan.yml`) is green for release commit. | `@maintainer-<name>` | ⬜ Pending | Workflow URL + run ID + commit SHA + final conclusion. |
| G3.5 | Manual/Maintainer required | Release assets published with checksums/signing artifacts (`*.sig`, `*.pem`) and verification output. | `@maintainer-<name>` | ⬜ Pending | Release asset links + verification command snippet + result. |
| G3.6 | Manual/Maintainer required | Optional installer/update artifacts (if shipped) validated and portable mode confirmed working. | `@maintainer-<name>` | ⬜ Pending | Platform/scenario matrix + evidence links + result. |

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Execution type | Verifiable criteria | Owner | Status | Evidence requirement |
| --- | --- | --- | --- | --- | --- |
| G4.1 | Automatable by Codex | `docs/RELEASE_QA_CHECKLIST.md` labels each checklist section/item as automatable vs manual and defines evidence payloads. | Codex | ✅ Complete (2026-03-03) | Checklist includes section ownership matrix and item-level execution/evidence columns. |
| G4.2 | Automatable by Codex | This document defines objective gate-based GO/NO-GO criteria tied to blocking items. | Codex | ✅ Complete (2026-03-03) | GO/NO-GO section includes blocker rule + blocker ID list. |
| G4.3 | Manual/Maintainer required | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | `@maintainer-<name>` | ⬜ Pending | Reviewer name + date + approval note/link. |
| G4.4 | Manual/Maintainer required | Release notes generated/reviewed and deferred non-blocking follow-ups tracked as issues. | `@maintainer-<name>` | ⬜ Pending | Release-notes artifact URL + sign-off + issue links/milestones. |

---

## Evidence Register (This Run — 2026-03-03)

| Evidence ID | Execution type | Command(s) | Outcome | Supports gates |
| --- | --- | --- | --- | --- |
| E1 | Automatable by Codex | `cd mama && go test ./...` | Pass | G1.1, G1.2, G1.3, G2.1, G2.2 |
| E2 | Automatable by Codex | `cd mama && go mod verify` | Pass | G3.1 |
| E3 | Automatable by Codex | `bash -n scripts/release/generate-checksums.sh` + checksum smoke flow | Pass | G3.2 |

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

### Current decision snapshot (from this run)

- **Decision:** `NO-GO (expected in-repo pre-release state)`
- **Automatable gates summary:** 8/8 complete (`G1.1-G1.3`, `G2.1-G2.2`, `G3.1-G3.2`, `G4.1-G4.2`).
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
