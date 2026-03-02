# v1.0 Readiness Review and Acceptance Criteria

This document defines objective acceptance gates required before publishing `v1.0.0`.

## Review Scope

A `v1.0.0` candidate is acceptable only when **all blocking gates** are complete with evidence.

Evidence must be one of:
- reproducible command output captured in this document (for automatable checks);
- linked maintainer evidence artifact (for manual checks).

---

## Acceptance Gates

### Gate 1 — Product functionality baseline (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G1.1 | Automatable by Codex | `cd mama && go test ./...` completes successfully. | ✅ Complete (2026-03-02) | `E1` command output block must show `ok`/`? [no test files]` only and no failing package. |
| G1.2 | Automatable by Codex | Config compatibility behavior remains validated (`mama/internal/config` package included in suite). | ✅ Complete (2026-03-02) | `E1` output includes `ok  mama/internal/config`. |
| G1.3 | Automatable by Codex | API/runtime compatibility behavior remains validated (`mama/cmd/mama` package included in suite). | ✅ Complete (2026-03-02) | `E1` output includes `ok  mama/cmd/mama`. |
| G1.4 | Manual/Maintainer required | Representative hardware flow succeeds: detect board → test port → map knobs → save → verify output changes. | ⬜ Pending | Owner placeholder + dated hardware run log with board model, firmware revision, host OS, serial port, observed behavior, and result (`PASS`/`FAIL`). |

### Gate 2 — Reliability and resilience (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G2.1 | Automatable by Codex | Protocol compatibility handling remains covered by tests (`mama/internal/proto` included in suite). | ✅ Complete (2026-03-02) | `E1` output includes `ok  mama/internal/proto`. |
| G2.2 | Automatable by Codex | Reconnect/backoff/metrics behavior remains covered by tests (`mama/internal/runtime` included in suite). | ✅ Complete (2026-03-02) | `E1` output includes `ok  mama/internal/runtime`. |
| G2.3 | Manual/Maintainer required | Long-run soak for release candidate completed according to `docs/SOAK_TEST_PLAN.md`. | ⬜ Pending | Owner placeholder + artifact bundle path/URL + run duration + explicit pass/fail summary. |

### Gate 3 — Platform and release quality (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G3.1 | Automatable by Codex | Dependency verification passes (`go mod verify` exits 0 with verified output). | ✅ Complete (2026-03-02) | `E2` output includes `all modules verified`. |
| G3.2 | Automatable by Codex (artifact preflight) | Release checksum script parses and smoke-checksum flow succeeds on staged files. | ✅ Complete (2026-03-02) | `E3` output includes checksum generation path and `OK` verification lines. |
| G3.3 | Manual/Maintainer required | CI matrix (`.github/workflows/ci.yml`) is green for release commit on Linux/Windows/macOS. | ⬜ Pending | Owner placeholder + workflow URL + run ID + commit SHA + final conclusion. |
| G3.4 | Manual/Maintainer required | Security scan (`.github/workflows/security-scan.yml`) is green for release commit. | ⬜ Pending | Owner placeholder + workflow URL + run ID + commit SHA + final conclusion. |
| G3.5 | Manual/Maintainer required | Release assets published with checksums/signing artifacts (`*.sig`, `*.pem`) and verification output. | ⬜ Pending | Owner placeholder + release asset links + verification command snippet + result. |
| G3.6 | Manual/Maintainer required | Optional installer/update artifacts (if shipped) validated and portable mode confirmed working. | ⬜ Pending | Owner placeholder + matrix rows (platform, scenario, result) + evidence links. |

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G4.1 | Automatable by Codex | `docs/RELEASE_QA_CHECKLIST.md` labels each checklist section and item as automatable or manual and defines evidence format. | ✅ Complete (2026-03-02) | Checklist section headers + rows include execution type and evidence format requirements. |
| G4.2 | Automatable by Codex | This review document defines objective gate-based GO/NO-GO criteria tied to blocking items. | ✅ Complete (2026-03-02) | GO/NO-GO section includes explicit blocker rule and current blocker IDs. |
| G4.3 | Manual/Maintainer required | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | ⬜ Pending | Owner placeholder + reviewer name + date + approval note/link. |
| G4.4 | Manual/Maintainer required | Release notes generated/reviewed and deferred non-blocking follow-ups tracked as issues. | ⬜ Pending | Owner placeholder + release-notes artifact URL + sign-off + issue links/milestones. |

---

## Evidence Captured This Run (2026-03-02)

### E1 — Host test suite (Automatable by Codex)

```bash
$ cd mama && go test ./...
ok  	mama/cmd/mama	0.018s
?   	mama/cmd/mama-ui	[no test files]
ok  	mama/internal/audio	0.019s
ok  	mama/internal/config	0.051s
ok  	mama/internal/proto	0.014s
ok  	mama/internal/runtime	0.014s
?   	mama/internal/serial	[no test files]
ok  	mama/internal/ui	0.024s
```

### E2 — Module dependency verification (Automatable by Codex)

```bash
$ cd mama && go mod verify
all modules verified
```

### E3 — Release checksum script preflight + smoke verification (Automatable by Codex)

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
