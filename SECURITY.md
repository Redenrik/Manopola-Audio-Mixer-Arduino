# Security Policy

## Supported Versions

The project is currently pre-`v1.0`.
Security fixes are provided on a best-effort basis for the latest `main` branch state.

| Version/Branch | Supported |
| --- | --- |
| `main` | Yes |
| Tagged pre-`v1.0` releases | Best effort (upgrade to latest recommended) |
| `v1.0+` (future) | Policy will be updated at first stable release |

## Reporting a Vulnerability

Do not open a public issue for suspected vulnerabilities.

Preferred reporting channels:

- GitHub Security Advisories: `Security -> Advisories -> Report a vulnerability`
- Email fallback: `security@manopola.dev`

Include:

- affected component(s) and version/commit
- reproduction steps or PoC
- impact assessment (confidentiality/integrity/availability)
- tested mitigations/patches
- whether public credit is requested

## Disclosure and Response Process

Target response windows:

- acknowledgement: within 3 business days
- triage decision: within 7 business days
- mitigation/fix plan: as soon as practical based on severity

Process:

1. Confirm receipt and start private triage.
2. Reproduce and classify severity.
3. Prepare and validate fix/mitigation.
4. Coordinate disclosure timing with reporter.
5. Publish advisory/release notes and credit (if requested).

## Scope Notes

In-scope areas:

- Host runtime (`mama/`) input parsing and serial handling
- Setup UI/API and config persistence paths
- Build/release artifact integrity process

Out-of-scope by default (unless chained with in-scope impact):

- purely physical attacks requiring direct device access
- custom local-environment misconfiguration
