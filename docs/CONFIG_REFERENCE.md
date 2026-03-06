# Configuration Reference (`config.yaml`)

MAMA stores runtime and mapping settings in YAML.

Default resolution order:

1. `./config.yaml`
2. `mama/internal/config/default.yaml`

## Minimal Example

```yaml
serial:
  port: "COM3"
  baud: 115200

debug: false

mappings:
  - knob: 1
    target: master_out
    step: 0.02
```

## Top-Level Fields

- `serial.port` (string, required): serial device path (`COM3`, `/dev/ttyACM0`, ...)
- `serial.baud` (int, optional): default `115200`
- `debug` (bool, optional): verbose logs
- `mappings` (array): active mapping set when profiles are not used
- `groups` (array, optional): reusable named selector sets
- `templates` (array, optional): saved mapping templates
- `default_template` (string, optional): default template id in UI
- `profiles` (array, optional): named mapping profiles
- `active_profile` (string, optional): selected profile name

## Mapping Fields

Each `mappings[]` item:

- `knob` (int, required): must be unique and `> 0`
- `target` (required): one of:
  - `master_out`
  - `mic_in`
  - `line_in`
  - `app`
  - `group`
- `step` (float, required): `> 0` and `<= 1` (for example `0.02` = 2%)
- `sensitivity` (optional): `slow`, `normal`, or `fast`
- `priority` (optional int): precedence for overlapping `app/group` mappings

Selector fields by target:

- `app`: use `selector` (single object)
- `group`: use `selectors` (array of selector objects)
- system targets (`master_out`, `mic_in`, `line_in`): selector fields are not allowed

Fallback fields (allowed only when `target` is `app` or `group`):

- `fallback_target` (optional): same target enum as above
- `fallback_name` (required when fallback target is `app` or `group`)

Legacy compatibility:

- old `name` fields for `app/group` are migrated to selector form on load
- `fallback_to_master: true` is still read and normalized

## Selector Object

```yaml
kind: exe
value: discord
```

Allowed selector kinds:

- `exact`
- `contains`
- `prefix`
- `suffix`
- `glob`
- `exe`

Matching is case-insensitive after normalization.

## App and Group Examples

### App mapping

```yaml
- knob: 2
  target: app
  selector:
    kind: exe
    value: discord
  step: 0.02
```

### Group mapping

```yaml
- knob: 3
  target: group
  selectors:
    - kind: exe
      value: chrome
    - kind: exe
      value: msedge
  step: 0.02
```

### App mapping with fallback

```yaml
- knob: 4
  target: app
  selector:
    kind: exe
    value: spotify
  step: 0.02
  fallback_target: master_out
```

## Reusable Groups

Named group definitions:

```yaml
groups:
  - name: browser
    selectors:
      - kind: exe
        value: chrome
      - kind: exe
        value: firefox
```

## Templates

Custom template example:

```yaml
templates:
  - id: my-streaming
    name: Streaming
    mappings:
      - knob: 1
        target: master_out
        step: 0.02
      - knob: 2
        target: mic_in
        step: 0.02
```

## Profiles

Profile-based mapping sets:

```yaml
profiles:
  - name: Gaming
    mappings:
      - knob: 1
        target: master_out
        step: 0.02
  - name: Work
    mappings:
      - knob: 1
        target: app
        selector:
          kind: exe
          value: teams
        step: 0.02
active_profile: Gaming
```

If profiles exist and `active_profile` is empty, the first profile is used.

## Validation Rules (Important)

- `serial.port` must be non-empty
- no duplicate knob IDs in the active mapping set
- each mapping `step` must be in `(0, 1]`
- `app` needs `selector`
- `group` needs non-empty `selectors`
- overlapping `app/group` selectors with equal precedence are rejected

## Runtime Notes

- Turning a knob up (`delta > 0`) auto-unmutes the same target.
- App/group control depends on active audio sessions.
- Discovery lists in UI are runtime snapshots; unavailable targets can still be configured by selector.
