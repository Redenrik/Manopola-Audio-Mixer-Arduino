# v1.0 Readiness Review and Acceptance Criteria

This document defines the acceptance criteria required before publishing `v1.0.0`.

## Review Scope

A `v1.0.0` candidate is acceptable only when all criteria below are marked complete with evidence links (workflow runs, logs, artifacts, or issue references).

## Acceptance Criteria

### 1) Product functionality baseline

- [ ] Core targets operate as documented: `master_out`, `mic_in`, `line_in`, `app`, and `group`.
- [ ] Setup UI first-run flow works end-to-end: detect board, test port, map knobs, save, verify.
- [ ] Config compatibility aliases load correctly without breaking current schema precedence.
- [ ] `/api/targets` returns backward-compatible fields (`known`, `supported`) and discovery metadata (`discovered`).

Evidence:
- Test output from `cd mama && go test ./...`.
- Manual verification notes for one representative hardware setup.

### 2) Reliability and resilience

- [ ] Protocol version handshake behavior is validated (`V:1` accepted; mismatch events safely dropped/logged).
- [ ] Serial reconnect behavior confirms bounded backoff and clean recovery.
- [ ] Runtime metrics and structured logs are emitted for parse errors, drops, reconnects, and backend failures.
- [ ] Long-run soak process is executed per `docs/SOAK_TEST_PLAN.md` and reviewed.

Evidence:
- Runtime logs captured during replay and hardware smoke tests.
- Soak artifacts under `artifacts/soak/` (or attached CI artifacts).

### 3) Platform and release quality gates

- [ ] CI matrix is green on Linux, Windows, and macOS.
- [ ] Security scan workflow is green (`go mod verify`, tidy drift check, `govulncheck`).
- [ ] Release artifacts include checksums and signature assets (`*.sig`, `*.pem`).
- [ ] Optional installer/update artifacts (if shipped) are validated and portable mode remains supported.

Evidence:
- Workflow links for `.github/workflows/ci.yml`, `.github/workflows/security-scan.yml`, `.github/workflows/release-artifacts.yml`, and `.github/workflows/release-signing.yml`.
- Attached checksum manifest and verification output.

### 4) Documentation and governance readiness

- [ ] `docs/RELEASE_QA_CHECKLIST.md` is completed for the release candidate.
- [ ] Support/deprecation and security policies are reviewed (`docs/SUPPORT_POLICY.md`, `SECURITY.md`).
- [ ] Release notes are generated/reviewed and include upgrade guidance.
- [ ] Any deferred non-blocking follow-ups are captured as GitHub issues with milestone labels.

Evidence:
- Completed release checklist in PR/release notes.
- Links to created follow-up issues (if any).

## Exit Decision

Record one of the following outcomes in the release PR:

- **GO**: all acceptance criteria complete with evidence.
- **NO-GO**: one or more blocking criteria incomplete; include remediation owner and target date.

## Suggested Sign-off Template

```text
v1.0 readiness decision: GO | NO-GO
Date:
Release candidate commit/tag:
Reviewers:
Blocking gaps (if any):
Evidence links:
```
