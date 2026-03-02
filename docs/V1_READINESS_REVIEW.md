# v1.0 Readiness Review and Acceptance Criteria

This document defines objective acceptance gates required before publishing `v1.0.0`.

## Review Scope

A `v1.0.0` candidate is acceptable only when **all blocking gates** are complete with evidence. Evidence must be either:
- a reproducible command result captured in this document; or
- a linked manual sign-off artifact (issue/PR comment, workflow URL, screenshot/log attachment, or release asset URL).

---

## Acceptance Gates

### Gate 1 — Product functionality baseline (Blocking)

| ID | Criteria (verifiable) | Ownership | Status | Required evidence format |
| --- | --- | --- | --- | --- |
| G1.1 | Host code/tests compile and pass all module tests. | Automatable by Codex | ✅ Complete | Dated command block for `cd mama && go test ./...`. |
| G1.2 | Config compatibility aliases continue loading without current-schema precedence regressions. | Automatable by Codex | ✅ Complete | Covered by `go test` (`mama/internal/config` package) result in command block. |
| G1.3 | Backward-compatible `/api/targets` fields (`known`, `supported`) and discovery metadata (`discovered`) remain validated in tests. | Automatable by Codex | ✅ Complete | Covered by `go test` (`mama/cmd/mama` + API tests) result in command block. |
| G1.4 | Representative hardware flow succeeds end-to-end (detect board → test port → map knobs → save → verify output changes). | Manual/Maintainer required | ⬜ Pending | Owner + dated test log (board model, host OS, serial port, observed behavior). |

#### Evidence captured this run (2026-03-02)

```bash
$ cd mama && go test ./...
ok   	mama/cmd/mama	0.017s
?    	mama/cmd/mama-ui	[no test files]
ok   	mama/internal/audio	0.020s
ok   	mama/internal/config	0.037s
ok   	mama/internal/proto	0.014s
ok   	mama/internal/runtime	0.015s
?    	mama/internal/serial	[no test files]
ok   	mama/internal/ui	0.024s
```

Manual sign-off placeholder:
- **Owner:** `@maintainer-<name>`
- **Evidence link:** `<PR comment / issue / artifact URL>`

---

### Gate 2 — Reliability and resilience (Blocking)

| ID | Criteria (verifiable) | Ownership | Status | Required evidence format |
| --- | --- | --- | --- | --- |
| G2.1 | Protocol compatibility handling (`V:1` accepted; mismatches dropped/logged) remains covered by automated tests. | Automatable by Codex | ✅ Complete | `go test` output including `mama/internal/proto` + `mama/cmd/mama`. |
| G2.2 | Reconnect + bounded backoff + runtime metrics/logging paths remain validated by automated tests. | Automatable by Codex | ✅ Complete | `go test` output including `mama/internal/runtime` + `mama/cmd/mama`. |
| G2.3 | Long-run soak execution completed per `docs/SOAK_TEST_PLAN.md` for release candidate build. | Manual/Maintainer required | ⬜ Pending | Owner + dated artifact bundle path (logs/metrics) and pass/fail note. |

Manual sign-off placeholder:
- **Owner:** `@maintainer-<name>`
- **Evidence link:** `<soak artifact URL/path + summary>`

---

### Gate 3 — Platform and release quality (Blocking)

| ID | Criteria (verifiable) | Ownership | Status | Required evidence format |
| --- | --- | --- | --- | --- |
| G3.1 | Module dependency verification passes. | Automatable by Codex | ✅ Complete | Dated command block for `cd mama && go mod verify`. |
| G3.2 | CI matrix (`.github/workflows/ci.yml`) green for release commit on Linux/Windows/macOS. | Manual/Maintainer required | ⬜ Pending | Workflow URL + run ID + commit SHA. |
| G3.3 | Security scan (`.github/workflows/security-scan.yml`) green for release commit. | Manual/Maintainer required | ⬜ Pending | Workflow URL + run ID + commit SHA. |
| G3.4 | Release artifacts published with checksums and signing assets (`*.sig`, `*.pem`) and checksum verification output. | Manual/Maintainer required | ⬜ Pending | Release asset links + verification command output snippet. |
| G3.5 | Optional installer/update artifacts (if shipped) validated; portable mode still functional. | Manual/Maintainer required | ⬜ Pending | Owner statement + test matrix table + evidence links. |

#### Evidence captured this run (2026-03-02)

```bash
$ cd mama && go mod verify
all modules verified
```

Manual sign-off placeholder:
- **Owner:** `@maintainer-<name>`
- **Evidence link:** `<workflow/release URLs>`

---

### Gate 4 — Documentation and governance readiness (Blocking)

| ID | Criteria (verifiable) | Ownership | Status | Required evidence format |
| --- | --- | --- | --- | --- |
| G4.1 | `docs/RELEASE_QA_CHECKLIST.md` is completed for the candidate and attached to release PR. | Manual/Maintainer required | ⬜ Pending | Completed checklist section in PR description or linked artifact. |
| G4.2 | Support/security policy review recorded against `docs/SUPPORT_POLICY.md` and `SECURITY.md`. | Manual/Maintainer required | ⬜ Pending | Reviewer name + dated approval note/link. |
| G4.3 | Release notes generated and reviewed, including upgrade guidance. | Manual/Maintainer required | ⬜ Pending | Generated notes artifact URL + reviewer sign-off. |
| G4.4 | Deferred non-blocking follow-ups tracked as issues with milestone labels. | Manual/Maintainer required | ⬜ Pending | Issue links + milestone names. |

Manual sign-off placeholder:
- **Owner:** `@maintainer-<name>`
- **Evidence link:** `<PR/release notes/issue URLs>`

---

## Objective GO/NO-GO Decision

Use these gates to make the decision:

- **GO** only if **all Blocking items (G1.x–G4.x)** are complete with evidence.
- **NO-GO** if any blocking item is pending/failed; include remediation owner and target date.

### Current decision snapshot (from this run)

- **Decision:** `NO-GO (expected in-repo pre-release state)`
- **Reason:** Automatable in-repo checks passed, but manual/hardware/workflow/release sign-offs are still pending.
- **Next required sign-offs:** G1.4, G2.3, G3.2–G3.5, G4.1–G4.4.

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
