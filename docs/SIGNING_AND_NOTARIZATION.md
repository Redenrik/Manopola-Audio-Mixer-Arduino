# Signing and Notarization Workflow

This document defines how maintainers sign release artifacts and (optionally) notarize macOS app bundles.

## Scope

- **Artifact signing:** required for all published releases.
- **macOS notarization:** required only when shipping macOS `.app` bundles.

## Automation Overview

GitHub Actions workflow: `.github/workflows/release-signing.yml`

Trigger modes:
- automatic on `release.published`
- manual via `workflow_dispatch` with `release_tag`

Jobs:
1. `sign-release-artifacts` (Ubuntu)
   - downloads release assets
   - signs each asset with keyless Sigstore (`cosign sign-blob`)
   - uploads `*.sig` and `*.pem` files back to the release
2. `notarize-macos-artifacts` (macOS, conditional)
   - runs only when Apple notarization secrets are configured
   - codesigns and notarizes macOS `.app` bundles from release zip assets
   - staples notarization tickets and uploads notarized zips

## Required Secrets

For signing:
- no long-lived signing key is required (keyless Sigstore with OIDC).

For notarization (only if releasing macOS app bundles):
- `APPLE_DEVELOPER_ID_APP` (Developer ID Application identity)
- `APPLE_NOTARY_KEY_ID` (App Store Connect API key ID)
- `APPLE_NOTARY_ISSUER_ID` (App Store Connect issuer ID)
- `APPLE_NOTARY_PRIVATE_KEY` (API key contents, PEM/P8 text)

## Local Maintainer Commands

Sign all files in a release artifact directory:

```bash
scripts/release/sign-artifacts.sh dist/release-assets
```

Notarize macOS zip artifacts in a release directory:

```bash
APPLE_DEVELOPER_ID_APP="Developer ID Application: ..." \
APPLE_NOTARY_KEY_ID="..." \
APPLE_NOTARY_ISSUER_ID="..." \
APPLE_NOTARY_PRIVATE_KEY="$(cat AuthKey_XXXX.p8)" \
scripts/release/notarize-macos.sh dist/release-assets
```

## Verification

After release-signing workflow completes:
- verify each primary release asset has matching `.sig` and `.pem` files.
- verify checksum manifests (`SHA256SUMS-*.txt`) are present and signed.
- if macOS assets are shipped, verify notarized zip artifacts are attached.

## Notes

- Keep portable artifacts available even when installer/notarized variants are introduced.
- Signing and notarization augment (not replace) checksum verification.
