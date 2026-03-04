# Production Readiness

This is the operator checklist to validate a release candidate before publishing.

## Automated Gate

Run:

```bash
make release-readiness
```

Or directly:

```bash
bash scripts/release/production-readiness-check.sh
```

Windows PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release/production-readiness-check.ps1
```

Automated checks:
- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `go mod verify`
- `govulncheck ./...` (if installed)
- cross-build compile checks:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
  - `windows/amd64`
  - `windows/386`

Artifacts:
- `artifacts/readiness/<run_id>/summary.txt`
- per-check logs in the same folder

## CI Enforcement

Workflow:
- `.github/workflows/release-readiness.yml`

Trigger:
- `push` to `main`
- `pull_request`
- `workflow_dispatch`

## Manual Gates Before Release

Automated checks are necessary but not sufficient.
Maintainers should still verify:

1. Hardware validation on representative boards/OS combinations
2. Soak execution per [SOAK_TEST_PLAN.md](SOAK_TEST_PLAN.md)
3. Release assets are published for all supported user targets
4. Checksums/signatures are attached and verifiable
5. macOS notarization evidence when shipping notarized macOS assets

## User-Impact Sanity Pass

Before publishing, run a quick UX sanity pass:
- fresh install on Windows 64-bit and 32-bit
- first-launch mapping save flow works
- tray behavior works (close-to-tray, reopen, quit)
- serial auto-detection and manual selection both work
