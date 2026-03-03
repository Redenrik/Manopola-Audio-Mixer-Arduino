# Troubleshooting

## Setup UI cannot detect serial ports

- Ensure Arduino master board is connected via USB.
- Press `Refresh` in the setup UI.
- Check that no other app is locking the COM port.
- Manually type the port if auto-detection fails.

## Identify mode does not flash knob indicators

- Confirm correct serial port and baud (`115200`).
- Verify master firmware is flashed and running.
- Verify I2C wires between master/slave (SDA/SCL + GND + 5V).
- Check setup UI status line for stream errors.

## Runtime starts but knob does nothing

- Open `config.yaml` and verify knob mapping exists.
- Ensure mapping `step` is greater than 0.
- Use `master_out` universally, or `mic_in` / `line_in` on hosts that expose capture controls (`pactl`/`amixer`). `app` mappings are supported on Unix hosts with `pactl` sink-input controls.
- Enable `debug: true` and read daemon logs for parse/mapping errors.
- Runtime logs are now structured as deterministic `event=<name> key=value ...` entries (for example: `event=serial_state port="COM3" state="connected"`).
- Check `event=runtime_metrics` log lines for aggregate counters (`parse_errors`, `dropped_events`, `reconnect_count`, `backend_failures`) while troubleshooting noisy serial links or backend issues.

## Runtime exits with config error

Common causes:
- missing `serial.port`
- duplicate knob IDs
- invalid `step` (must be `> 0` and `<= 1`)
- `app` mapping missing `selector` (or invalid selector kind/value)
- `group` mapping missing `selectors`
- overlapping `app/group` mappings with identical precedence (set distinct `priority` values)
- group selector set matched no active app sessions (ensure target apps are running and selector kinds/values match)
- negative mapping `priority`

## Press action does not mute

- Press action emits `B<id>:1` only.
- Release events are intentionally ignored.
- Confirm your mapping target is currently supported (`master_out`, plus `mic_in` / `line_in` when capture tooling is available, and `app` on Unix hosts with `pactl`).

## `scripts/quickstart.sh` fails immediately

Common causes and fixes:
- `error: Go toolchain not found in PATH...` -> install Go and retry (`go version` should work).
- `expected Go module directory ... was not found` -> run the script from this repository checkout (do not copy script alone).
- Permission denied on launchers -> ensure files are executable (`chmod +x dist/mama-quickstart/*.sh`).

## Security scan commands fail with `403 Forbidden`

If `go mod tidy` / `go install ... govulncheck` cannot download modules, your network/proxy may block outbound access to `proxy.golang.org` or GitHub.

- Validate connectivity or set your corporate `GOPROXY`.
- Re-run:
  - `cd mama && go mod tidy && git diff --exit-code -- go.mod go.sum`
  - `cd mama && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`

