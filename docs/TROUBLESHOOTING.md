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
- For now, use `master_out` target only (other targets are planned).
- Enable `debug: true` and read daemon logs for parse/mapping errors.

## Runtime exits with config error

Common causes:
- missing `serial.port`
- duplicate knob IDs
- invalid `step` (must be `> 0` and `<= 1`)
- `app/group` mapping missing `name`

## Press action does not mute

- Press action emits `B<id>:1` only.
- Release events are intentionally ignored.
- Confirm your mapping target is currently supported (`master_out`).
