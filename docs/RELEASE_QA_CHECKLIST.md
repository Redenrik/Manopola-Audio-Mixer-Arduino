# Release QA Checklist

Use this checklist before tagging or publishing a release artifact.

## 1) Build & Test Baseline

- [ ] CI is green for `.github/workflows/ci.yml` and `.github/workflows/security-scan.yml` on the release commit.
- [ ] Local verification: `cd mama && go test ./...`.
- [ ] Local verification: `cd mama && go mod verify`.
- [ ] No unreviewed changes in `go.mod`/`go.sum`.

## 2) Runtime and Config Verification

- [ ] Setup UI loads and can save a valid config file.
- [ ] Serial connection test succeeds against a representative board.
- [ ] Knob rotation updates `master_out` volume and button toggles mute.
- [ ] Protocol compatibility behavior is validated (`V:1` accepted, mismatches logged and controls dropped).

## 3) Release Artifacts and Integrity

- [ ] Artifacts built for intended release platforms.
- [ ] SHA-256 checksums generated with `scripts/release/generate-checksums.sh`.
- [ ] Release assets signed (`*.sig` + `*.pem`) via `.github/workflows/release-signing.yml` or `scripts/release/sign-artifacts.sh`.
- [ ] If macOS `.app` artifacts are shipped, notarized/stapled zips are attached to the release.
- [ ] Checksum manifest verified (`sha256sum -c` or `shasum -a 256 -c`).
- [ ] Portable mode validated (binaries + `config.yaml` side-by-side, no service install required).
- [ ] Optional installer artifacts (if shipped) install/uninstall cleanly and preserve portable fallback availability.
- [ ] Optional `*-update-manifest.json` assets (if shipped) match published artifact URL/checksum/size.

## 4) Documentation and Governance

- [ ] Changelog / release notes generated via `.github/workflows/release-notes.yml` (or `scripts/release/generate-release-notes.sh`) and reviewed.
- [ ] Support/deprecation implications reviewed against `docs/SUPPORT_POLICY.md`.
- [ ] Security-impacting changes reviewed against `SECURITY.md` disclosure expectations.
- [ ] Open TODOs and follow-ups captured as GitHub issues.
- [ ] `docs/V1_READINESS_REVIEW.md` acceptance criteria reviewed and decision recorded (`GO`/`NO-GO`).

## 5) Sign-off

- [ ] Final release candidate approved by maintainers.
- [ ] Tag created and release published with checksums attached.
