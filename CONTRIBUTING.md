# Contributing to MAMA

## Scope

MAMA combines embedded firmware and desktop software. Changes often affect both:
- `firmware/*` for hardware event generation
- `mama/*` for serial parsing, mapping, and audio control

When submitting a PR, clearly state which layer is changed and why.

## Development Setup

### Go app

```bash
cd mama
go test ./...
go run ./cmd/mama-ui
go run ./cmd/mama
```

### Firmware

- `firmware/master/master.ino` -> flash to USB-connected board.
- `firmware/slave/slave.ino` -> flash to I2C slave board.
- Confirm baud is `115200` and I2C address is `0x12`.

## PR Checklist

- Keep behavior changes small and explicit.
- Add/update tests for config/protocol/logic changes.
- Update docs when behavior or user flow changes.
- Preserve protocol compatibility unless explicitly versioning it.
- Verify no hard-coded machine-specific paths or secrets are added.

## Code Conventions

- Go:
  - run `gofmt` on changed files
  - prefer small packages with explicit boundaries
  - keep error messages actionable
- Arduino:
  - avoid dynamic allocation
  - keep ISR/I2C handlers minimal and deterministic
  - document protocol changes inline

## Branch & Commit Guidance

- Branch names:
  - `feat/<topic>`
  - `fix/<topic>`
  - `docs/<topic>`
- Commit style:
  - one logical change per commit
  - imperative subject line
  - include impacted area (`firmware`, `daemon`, `ui`, `docs`)

## Testing Expectations

Minimum for code PRs:
- `go test ./...`
- manual smoke test:
  - config load/save
  - serial parsing from real board or captured log
  - `master_out` knob rotation + mute press

## Security and Safety Notes

- Do not auto-run privileged actions.
- Do not install background services silently.
- Keep default operation portable and reversible.
- Reject malformed serial events safely (already enforced in parser).
