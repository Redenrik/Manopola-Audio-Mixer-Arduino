# Production Readiness

This document is the practical operator guide for moving MAMA from feature-complete development to a releasable production state.

For formal release gate ownership and sign-off:
- [V1_READINESS_REVIEW.md](V1_READINESS_REVIEW.md)
- [RELEASE_QA_CHECKLIST.md](RELEASE_QA_CHECKLIST.md)

## Automated Gate (Codex/CI)

Run the production readiness gate locally:

```bash
make release-readiness
```

Or directly:

```bash
bash scripts/release/production-readiness-check.sh
```

Windows PowerShell equivalent:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release/production-readiness-check.ps1
```

Checks executed:
- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `go mod verify`
- `govulncheck ./...` (when installed)
- cross-build compile checks:
  - `GOOS=linux GOARCH=amd64`
  - `GOOS=darwin GOARCH=amd64`
  - `GOOS=windows GOARCH=amd64`

Artifacts:
- per-run logs and summary are stored under `artifacts/readiness/<run_id>/`.

## CI Enforcement

Workflow:
- `.github/workflows/release-readiness.yml`

Trigger:
- `push` to `main`
- `pull_request`
- `workflow_dispatch`

The workflow runs the gate on Linux, Windows, and macOS and uploads per-platform readiness artifacts for audit/debug.

## Manual Production Gates (Maintainers)

Automated checks alone are not enough for release.

Required manual evidence before publication:
- hardware validation on representative boards/OS combinations
- soak execution per [SOAK_TEST_PLAN.md](SOAK_TEST_PLAN.md)
- signed and verified release artifacts
- macOS notarization evidence when shipping macOS binaries
- release notes and support/security policy review sign-off

Use [RELEASE_QA_CHECKLIST.md](RELEASE_QA_CHECKLIST.md) as the single source of truth for manual owner-assigned evidence.

## Supportability Baseline

The setup UI can export a diagnostics JSON bundle (`Export Diagnostics`) containing:
- runtime status snapshot
- targets/mapping-status API snapshots
- active in-form config
- platform/browser metadata
- recent debug log tail

Use that bundle for bug reports and release-candidate triage.
