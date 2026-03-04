# Long-Run Soak Test Plan

This plan validates host runtime stability over extended runs and produces deterministic artifacts for regression review.

## Goals

- Verify repeated test execution remains stable without intermittent failures.
- Validate runtime/protocol packages (`internal/runtime`, `internal/proto`) stay reliable under repeated execution.
- Produce timestamped artifacts that can be attached to CI runs or manual validation reports.

## Scope

Included in this soak plan:
- Host Go test suite (`go test ./...`).
- Focused runtime/protocol test loop (`go test ./internal/runtime ./internal/proto`).

Not included:
- Hardware-in-the-loop serial endurance tests (tracked separately).
- Firmware-side encoder/I2C stress tests (tracked in firmware test scripts).

## Scripted Verification Artifact

Use:

```bash
scripts/soak/run_host_soak.sh [iterations]
```

Defaults:
- `iterations=25` if omitted.

Behavior:
- Creates `artifacts/soak/<UTC timestamp>/`.
- Runs two checks per iteration:
  1. `go test ./...`
  2. `go test ./internal/runtime ./internal/proto`
- Writes per-check logs and a `summary.txt` file.
- Stops immediately on first failure and marks `result=FAIL`.

## Artifact Review Checklist

After a run:

1. Confirm `summary.txt` ends with `result=PASS`.
2. Confirm no `status=FAIL` lines are present in `summary.txt`.
3. If failed, open the referenced log file and capture:
   - failing command,
   - iteration number,
   - exact test failure output.

## Suggested Cadence

- Before release candidates: run at least `50` iterations.
- For routine maintenance: run at least `10` iterations after runtime/protocol changes.
- For incident follow-up: run targeted soak with `25+` iterations after fixes.
