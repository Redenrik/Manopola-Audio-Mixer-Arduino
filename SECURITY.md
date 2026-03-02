# Security Policy

## Supported Versions

This project is currently in pre-`v1.0` development, and security fixes are provided on a best-effort basis for the latest commit on `main`.

| Version/Branch | Supported |
| --- | --- |
| `main` | ✅ |
| Tagged pre-`v1.0` releases | ⚠️ Best effort (may require upgrading to `main`) |
| `v1.0+` (future) | Policy will be updated at first stable release |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for suspected vulnerabilities.

Report privately via one of these channels:
- GitHub Security Advisories (preferred): use the repository **Security > Advisories > Report a vulnerability** flow.
- Email fallback: `security@manopola.dev`.

Include as much detail as possible:
- Affected component(s) and version/commit.
- Reproduction steps or proof-of-concept.
- Impact assessment (confidentiality/integrity/availability).
- Any mitigations or patches you already validated.
- Whether public credit is desired.

## Disclosure and Response Process

Maintainers target the following response times:
- **Acknowledgement:** within 3 business days.
- **Triage decision:** within 7 business days.
- **Fix or mitigation plan:** as soon as practical based on severity and release risk.

Process:
1. Confirm report receipt and begin private triage.
2. Reproduce and classify severity.
3. Prepare and validate a fix/mitigation.
4. Coordinate disclosure timing with the reporter.
5. Publish advisory/release notes and credit reporter (if requested).

## Scope Notes

In-scope areas include:
- Host runtime (`mama/`) input parsing and serial protocol handling.
- Setup UI HTTP endpoints and config persistence paths.
- Build/release artifact integrity processes.

Out-of-scope (unless chained with an in-scope impact):
- Physically local attacks requiring direct device access.
- Misconfiguration in custom local environments.
