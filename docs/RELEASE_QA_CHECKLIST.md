# Release QA Checklist

Use this checklist before tagging or publishing a release artifact.

> Status legend: `✅ Complete`, `⬜ Pending`, `❌ Failed`, `⚠️ Blocked`

---

## 1) Build & Test Baseline

### 1.1 Automatable by Codex

| ID | Verification step | Command/reference | Current status | Evidence |
| --- | --- | --- | --- | --- |
| B1 | Run full host test suite from module root. | `cd mama && go test ./...` | ✅ Complete (2026-03-02) | See evidence block below. |
| B2 | Verify module dependency checksums and cache integrity. | `cd mama && go mod verify` | ✅ Complete (2026-03-02) | See evidence block below. |

Evidence captured this run:

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

$ cd mama && go mod verify
all modules verified
```

### 1.2 Manual/Maintainer required

| ID | Verification step | Required owner evidence |
| --- | --- | --- |
| B3 | Confirm CI is green for `.github/workflows/ci.yml` and `.github/workflows/security-scan.yml` on the release commit. | Owner: `@maintainer-<name>`; include workflow URLs + run IDs + commit SHA. |
| B4 | Confirm no unreviewed `go.mod`/`go.sum` drift in release PR. | Owner: `@maintainer-<name>`; include PR review link/comment explicitly acknowledging dependency diff. |

---

## 2) Runtime and Config Verification

### 2.1 Automatable by Codex

| ID | Verification step | Command/reference | Current status | Evidence |
| --- | --- | --- | --- | --- |
| R1 | Validate protocol compatibility behavior remains covered by tests (`V:1` accept, mismatches dropped/logged). | Covered by `cd mama && go test ./...` (`mama/internal/proto`, `mama/cmd/mama`). | ✅ Complete (2026-03-02) | Test output in Section 1 evidence block. |
| R2 | Validate config compatibility behavior remains covered by tests. | Covered by `cd mama && go test ./...` (`mama/internal/config`). | ✅ Complete (2026-03-02) | Test output in Section 1 evidence block. |

### 2.2 Manual/Maintainer required

| ID | Verification step | Required owner evidence |
| --- | --- | --- |
| R3 | Setup UI loads and can save a valid config file in representative runtime environment. | Owner: `@maintainer-<name>`; include environment (OS/browser), steps executed, and saved `config.yaml` snippet/path. |
| R4 | Serial connection test succeeds against representative board. | Owner: `@maintainer-<name>`; include board model, serial port, firmware revision, and observed output log excerpt. |
| R5 | Knob rotation updates `master_out` volume and button toggles mute. | Owner: `@maintainer-<name>`; include manual test video/log/screenshot + timestamp. |

---

## 3) Release Artifacts and Integrity

### 3.1 Automatable by Codex (when release artifacts are present locally)

| ID | Verification step | Command/reference | Current status | Evidence |
| --- | --- | --- | --- | --- |
| A1 | Generate SHA-256 checksums for staged artifacts. | `scripts/release/generate-checksums.sh <artifact_dir>` | ⬜ Pending | Requires local release artifact directory. |
| A2 | Verify checksum manifest against artifacts. | `sha256sum -c checksums.txt` or `shasum -a 256 -c checksums.txt` | ⬜ Pending | Requires `checksums.txt` from A1 and staged release files. |

### 3.2 Manual/Maintainer required

| ID | Verification step | Required owner evidence |
| --- | --- | --- |
| A3 | Confirm intended platform artifacts were produced/published. | Owner: `@maintainer-<name>`; include release asset list URL and platform coverage note. |
| A4 | Confirm release assets signed (`*.sig` + `*.pem`) via workflow/script. | Owner: `@maintainer-<name>`; include workflow run URL or signing log + asset links. |
| A5 | If macOS `.app` artifacts are shipped, confirm notarized/stapled zips attached. | Owner: `@maintainer-<name>`; include notarization ticket/log and release asset links. |
| A6 | Confirm portable mode validation (binary + `config.yaml` side-by-side, no service install). | Owner: `@maintainer-<name>`; include executed commands and runtime result summary. |
| A7 | If optional installer artifacts are shipped, validate install/uninstall + portable fallback availability. | Owner: `@maintainer-<name>`; include installer test matrix and logs. |
| A8 | If `*-update-manifest.json` is shipped, validate URL/checksum/size fields match published assets. | Owner: `@maintainer-<name>`; include manifest diff/check output and release links. |

---

## 4) Documentation and Governance

### 4.1 Automatable by Codex

| ID | Verification step | Command/reference | Current status | Evidence |
| --- | --- | --- | --- | --- |
| D1 | Ensure readiness review doc exists with objective gates and GO/NO-GO criteria. | `docs/V1_READINESS_REVIEW.md` | ✅ Complete (2026-03-02) | Updated with objective blocking gates and decision logic. |

### 4.2 Manual/Maintainer required

| ID | Verification step | Required owner evidence |
| --- | --- | --- |
| D2 | Changelog/release notes generated and reviewed. | Owner: `@maintainer-<name>`; include generated notes artifact URL and review sign-off. |
| D3 | Support/deprecation implications reviewed against `docs/SUPPORT_POLICY.md`. | Owner: `@maintainer-<name>`; include reviewer name/date and approval note. |
| D4 | Security-impacting changes reviewed against `SECURITY.md`. | Owner: `@maintainer-<name>`; include security review note/link. |
| D5 | Open TODOs/follow-ups captured as GitHub issues. | Owner: `@maintainer-<name>`; include issue URLs + milestone labels. |
| D6 | `docs/V1_READINESS_REVIEW.md` decision recorded as GO/NO-GO for candidate commit/tag. | Owner: `@maintainer-<name>`; include completed sign-off block in release PR. |

---

## 5) Final Sign-off (Manual/Maintainer required)

| ID | Verification step | Required owner evidence |
| --- | --- | --- |
| S1 | Final release candidate approved by maintainers. | Owner: `@maintainer-<name>`; include explicit approval comment(s). |
| S2 | Tag created and release published with checksums attached. | Owner: `@maintainer-<name>`; include tag/release URL and checksum asset link. |
