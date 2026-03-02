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
| G1.1 | Automatable by Codex | Host module test suite passes. | ✅ Complete (2026-03-02) | Dated `cd mama && go test ./...` output block. |
| G1.2 | Automatable by Codex | Config compatibility behavior remains validated (`mama/internal/config` package included in suite). | ✅ Complete (2026-03-02) | Same dated `go test` output block showing `mama/internal/config`. |
| G1.3 | Automatable by Codex | API/runtime compatibility behavior remains validated (`mama/cmd/mama` package included in suite). | ✅ Complete (2026-03-02) | Same dated `go test` output block showing `mama/cmd/mama`. |
| G1.4 | Manual/Maintainer required | Representative hardware flow succeeds: detect board → test port → map knobs → save → verify output changes. | ⬜ Pending | Owner + dated test log (board model, firmware revision, host OS, serial port, observed behavior). |

### Gate 2 — Reliability and resilience (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G2.1 | Automatable by Codex | Protocol compatibility handling remains covered by tests (`mama/internal/proto` included in suite). | ✅ Complete (2026-03-02) | Dated `go test` output block showing `mama/internal/proto`. |
| G2.2 | Automatable by Codex | Reconnect/backoff/metrics behavior remains covered by tests (`mama/internal/runtime` included in suite). | ✅ Complete (2026-03-02) | Dated `go test` output block showing `mama/internal/runtime`. |
| G2.3 | Manual/Maintainer required | Long-run soak for release candidate completed according to `docs/SOAK_TEST_PLAN.md`. | ⬜ Pending | Owner + dated artifact bundle link/path + explicit pass/fail summary. |

### Gate 3 — Platform and release quality (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G3.1 | Automatable by Codex | Module dependency verification passes. | ✅ Complete (2026-03-02) | Dated `cd mama && go mod verify` output block. |
| G3.2 | Automatable by Codex (artifact preflight) | Release checksum script is syntactically valid and smoke-tested against a local artifact directory. | ✅ Complete (2026-03-02) | Dated command/output block for `bash -n scripts/release/generate-checksums.sh` and smoke run. |
| G3.3 | Manual/Maintainer required | CI matrix (`.github/workflows/ci.yml`) is green for release commit on Linux/Windows/macOS. | ⬜ Pending | Workflow URL + run ID + commit SHA. |
| G3.4 | Manual/Maintainer required | Security scan (`.github/workflows/security-scan.yml`) is green for release commit. | ⬜ Pending | Workflow URL + run ID + commit SHA. |
| G3.5 | Manual/Maintainer required | Release assets published with checksums/signing artifacts (`*.sig`, `*.pem`) and verification output. | ⬜ Pending | Release asset links + verification command output snippet. |
| G3.6 | Manual/Maintainer required | Optional installer/update artifacts (if shipped) validated and portable mode confirmed working. | ⬜ Pending | Owner statement + test matrix table + evidence links. |

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Execution type | Verifiable criteria | Status | Evidence requirement |
| --- | --- | --- | --- | --- |
| G4.1 | Automatable by Codex | `docs/RELEASE_QA_CHECKLIST.md` includes explicit execution type labels and evidence requirements for each checklist section. | ✅ Complete (2026-03-02) | Doc diff/reference in this PR. |
| G4.2 | Automatable by Codex | `docs/V1_READINESS_REVIEW.md` includes objective gate-based GO/NO-GO decision criteria tied to blocking items. | ✅ Complete (2026-03-02) | Doc diff/reference in this PR. |
| G4.3 | Manual/Maintainer required | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | ⬜ Pending | Reviewer name + dated approval note/link. |
| G4.4 | Manual/Maintainer required | Release notes generated/reviewed and deferred non-blocking follow-ups tracked as issues. | ⬜ Pending | Notes artifact URL + sign-off + issue links/milestones. |

---

## Evidence Captured This Run (2026-03-02)

### Automatable command evidence

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

### Manual evidence placeholders (must be completed by maintainer)

- **Owner:** `@maintainer-<name>`
- **Hardware validation evidence:** `<PR comment/issue artifact URL with required hardware log fields>`
- **CI/security workflow evidence:** `<workflow URLs + run IDs + commit SHA>`
- **Release/signing/notarization evidence:** `<release asset URLs + verification snippet>`
- **Governance/release notes evidence:** `<review links + notes artifact + follow-up issue links>`

---

## Objective GO/NO-GO Decision

- **GO** only if every blocking gate (G1.x through G4.x) is `✅ Complete` with attached evidence.
- **NO-GO** if any blocking gate is `⬜ Pending`, `❌ Failed`, or `⚠️ Blocked`; remediation owner + target date required.

### Current decision snapshot (from this run)

- **Decision:** `NO-GO (expected in-repo pre-release state)`
- **Objective blockers:** G1.4, G2.3, G3.3-G3.6, G4.3-G4.4.
- **Reason:** All in-repo automatable checks passed, while hardware/workflow/release/governance sign-offs remain manual and are still pending.

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
