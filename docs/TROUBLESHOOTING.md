# Troubleshooting

## Auto-detect does not select the right serial port

- Confirm the master board is connected and powered.
- Wait a few seconds after connecting USB before running auto-detect.
- Close tools that may lock serial ports (Arduino Serial Monitor, terminal monitors).
- Use manual port selection + **Test Connection** as fallback.

## Test Connection says OK but knobs do nothing

- Verify baud is `115200`.
- Confirm both boards are flashed (master + slave).
- Check I2C wiring between boards (SDA, SCL, 5V, GND).
- Enable debug logs and check for:
  - `event=serial_state`
  - `event=protocol_negotiated`
  - incoming `E...` / `B...` events

## Knob events arrive but target volume does not change

- Verify that knob has a mapping and valid `step` (> 0 and <= 1).
- For `app`/`group`, ensure target app sessions are active (currently producing audio).
- If display-name matching is unreliable, use selector token format:
  - `exe:discord`
  - `exact:Spotify`
  - `contains:chrome`

## App/group mapping says target unavailable

- This usually means no matching active audio session currently exists.
- Start audio playback in the target app.
- Use fallback target for resilience (for example `master_out` or another app/group).

## Startup-at-login does not work

- Windows: ensure `Start Mixer.cmd` exists next to `config.yaml` in your package folder.
- macOS: startup uses LaunchAgents and takes effect on next login.
- Linux: startup uses `~/.config/autostart/io.mama.mixer.desktop`; verify the file exists after applying startup setting.

## `start-mixer` says MAMA is already running

- Background launcher stores PID in `.mama.pid`.
- Use `Stop Mixer` / `stop-mixer.sh` first, then run `Start Mixer` / `start-mixer.sh` again.
- If process is no longer alive, delete `.mama.pid` and rerun.

## Runtime exits immediately with config error

Common causes:

- missing `serial.port`
- duplicate `knob` IDs
- invalid `step` (must be `> 0` and `<= 1`)
- `app` mapping without `selector`
- `group` mapping without `selectors`

Validate config using:

```bash
cd mama
go test ./internal/config
```

## Press action does not mute

- Buttons emit `B<id>:1` only.
- Confirm mapped target supports mute on your backend.
- Test with `master_out` first to isolate mapping vs endpoint issues.

## Windows installer choice is unclear

- Use `MAMA-Setup-Windows.exe` for 64-bit Windows.
- Use `MAMA-Setup-Windows-32bit.exe` for 32-bit Windows.

## Security tooling fails due network/proxy constraints

If module download fails:

- configure `GOPROXY` for your environment
- rerun:
  - `cd mama && go mod tidy && git diff --exit-code -- go.mod go.sum`
  - `cd mama && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`
