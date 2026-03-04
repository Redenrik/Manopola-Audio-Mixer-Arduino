# Troubleshooting

## Setup UI cannot detect serial ports

- Confirm Arduino master board is connected via USB.
- Click `Refresh` in setup.
- Ensure no other app is locking the COM/TTY port (Arduino Serial Monitor, terminal tools).
- Manually set port if auto-detect misses it.

## Port test says connected but knobs do not react

- Verify baud is `115200`.
- Confirm firmware is flashed on both master and slave.
- Check I2C wiring between boards (SDA/SCL + 5V + GND).
- Enable `debug: true` and inspect logs for `event=serial_state` and `event=serial_rx`.

## Knobs react but target volume does not change

- Open mappings and verify knob assignment exists.
- Check mapping `step` is greater than `0`.
- Ensure target exists and is active (especially for `app` and `group`).
- For app selectors, prefer `exe` kind when app display-name matching is unstable.

## Runtime starts then exits with config error

Common causes:
- missing `serial.port`
- duplicate knob IDs
- invalid `step` (`> 0` and `<= 1`)
- `app` mapping missing `selector`
- `group` mapping missing `selectors`
- overlapping `app/group` mappings with identical precedence (`priority` needed)

## Press action does not mute

- Buttons emit `B<id>:1` only.
- Confirm mapping target supports mute in your platform backend.
- Test with `master_out` first to isolate mapping vs target issues.

## Installer choice confusion on Windows

- Use `MAMA-Setup-Windows.exe` for 64-bit Windows.
- Use `MAMA-Setup-Windows-32bit.exe` for 32-bit Windows.

## Security scan fails with module download errors

If commands fail with proxy/network restrictions:
- configure `GOPROXY` for your environment
- rerun:
  - `cd mama && go mod tidy && git diff --exit-code -- go.mod go.sum`
  - `cd mama && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`
