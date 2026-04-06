# Gooey Config Reference

This document is the authoritative reference for Gooey's `config.json` file as implemented today.

It covers:

- every top-level section currently read by Gooey
- every field currently supported inside those sections
- the current built-in defaults
- the accepted value shapes and current runtime behavior

For the live implementation, see [pkg/components/config.go](/home/admin/gooey/pkg/components/config.go), [cmd/gooey/main.go](/home/admin/gooey/cmd/gooey/main.go), and [pkg/input/gpio.go](/home/admin/gooey/pkg/input/gpio.go).

## File Discovery

When Gooey starts, it looks for `config.json` in this order:

1. `GOOEY_CONFIG_PATH`
2. `./config.json`
3. `../config.json`
4. `../../config.json`
5. `/etc/gooey/config.json`

The first existing file is loaded.

If no file is found, Gooey uses built-in defaults in memory.

The calibration tool follows the same search order. If no config file exists, it creates `./config.json` and writes the current defaults before saving calibration results.

## Format

The file must be valid JSON.

Top-level shape:

```json
{
  "components": {
    "button": {
      "fontSize": 14,
      "textPadding": 4,
      "borderWidth": 2
    },
    "container": {},
    "label": {
      "fontSize": 13,
      "textPadding": 4
    },
    "toggle": {
      "textPadding": 3,
      "borderWidth": 2
    }
  },
  "touch": {
    "devicePath": "/dev/input/by-path/platform-fe204000.spi-cs-1-platform-stmpe-ts-event",
    "minXRaw": 387,
    "maxXRaw": 3520,
    "minYRaw": 642,
    "maxYRaw": 3523,
    "ScreenXPixels": 240,
    "ScreenYPixels": 320,
    "invertX": false,
    "invertY": true,
    "isLandscape": true
  },
  "gpio": {
    "buttons": [
      {
        "id": "button1",
        "pin": 23,
        "pull": "up",
        "invert": true
      }
    ]
  }
}
```

## Parsing Rules

- Unknown JSON object fields are ignored by the current Go unmarshal behavior.
- Missing sections fall back to built-in defaults.
- For many numeric override fields, `0` is treated the same as omitted because Gooey merges loaded values over defaults and only applies non-zero overrides.
- Boolean fields in the `touch` section always take the loaded value. In practice, omitted and `false` behave the same because the built-in defaults are also `false`.
- The `gpio.buttons` array is not merged item-by-item. If the file provides a non-empty `buttons` array, that array replaces the default list.

## Top-Level Fields

| Field | Type | Required | Default | Notes |
| --- | --- | --- | --- | --- |
| `components` | object | No | built-in defaults for built-in component types | Styling defaults by component type |
| `touch` | object | No | touch disabled with `/dev/null` and zero calibration | Touch device path and calibration |
| `gpio` | object | No | no GPIO buttons configured | GPIO-backed hardware inputs |

## `components`

The `components` object is a map from component type name to default styling.

Current built-in component types are:

- `container`
- `label`
- `button`
- `toggle`
- `soft_button_bar`

Example:

```json
"components": {
  "button": {
    "fontSize": 14,
    "textPadding": 4,
    "borderWidth": 2
  }
}
```

### Current Built-In Defaults

| Component Type | `fontSize` | `textPadding` | `borderWidth` |
| --- | --- | --- | --- |
| `container` | unset | unset | unset |
| `label` | `13` | `4` | unset |
| `button` | `14` | `6` | `2` |
| `toggle` | unset | `3` | `2` |
| `soft_button_bar` | `11` | `2` | `1` |

### Component Default Fields

| Field | Type | Valid Values | Current Behavior |
| --- | --- | --- | --- |
| `fontSize` | number | any JSON number | If non-zero, overrides the built-in default for that component type |
| `textPadding` | integer | any JSON integer | If non-zero, overrides the built-in default for that component type |
| `borderWidth` | integer | any JSON integer | If non-zero, overrides the built-in default for that component type |

Important current behavior:

- `0` does not override the built-in value. It is treated as omitted.
- There is currently no config-time validation for negative values here.
- Gooey only uses these defaults when a scene node does not explicitly provide its own style value.
- Additional component-type keys are accepted in JSON. They are only meaningful if Gooey has a component registered under the same type name.

## `touch`

The `touch` section configures the evdev touch device and the raw-to-screen coordinate transform.

Current built-in default:

```json
"touch": {
  "devicePath": "/dev/null",
  "minXRaw": 0,
  "maxXRaw": 0,
  "minYRaw": 0,
  "maxYRaw": 0,
  "ScreenXPixels": 0,
  "ScreenYPixels": 0,
  "invertX": false,
  "invertY": false,
  "isLandscape": false
}
```

### Touch Fields

| Field | Type | Valid Values | Default | Notes |
| --- | --- | --- | --- | --- |
| `devicePath` | string | any filesystem path string | `/dev/null` | Path to the Linux evdev touch device |
| `minXRaw` | integer | any JSON integer fitting in Go `int32` | `0` | Raw minimum X value used for scaling |
| `maxXRaw` | integer | any JSON integer fitting in Go `int32` | `0` | Raw maximum X value used for scaling |
| `minYRaw` | integer | any JSON integer fitting in Go `int32` | `0` | Raw minimum Y value used for scaling |
| `maxYRaw` | integer | any JSON integer fitting in Go `int32` | `0` | Raw maximum Y value used for scaling |
| `ScreenXPixels` | integer | any JSON integer | `0` | Screen width in the touch controller's native axis space |
| `ScreenYPixels` | integer | any JSON integer | `0` | Screen height in the touch controller's native axis space |
| `invertX` | boolean | `true` or `false` | `false` | Inverts the scaled X axis |
| `invertY` | boolean | `true` or `false` | `false` | Inverts the scaled Y axis |
| `isLandscape` | boolean | `true` or `false` | `false` | Swaps X and Y after scaling |

Important current behavior:

- The JSON field names for screen dimensions are currently case-sensitive and must be `ScreenXPixels` and `ScreenYPixels` exactly as shown above.
- Numeric touch fields only override the built-in defaults when non-zero. A value of `0` behaves like omission.
- Gooey currently does not validate min/max ordering at config load time.
- If Gooey cannot open the configured `devicePath`, it logs a warning and continues running without touch input.
- The calibration tool updates the `touch` section and writes the full config file back to disk.
- `isLandscape` is currently documented in code as not fully implemented in terms of changing drawing orientation. It is used in touch coordinate transformation.

### Touch Transform Order

Current runtime order in [pkg/input/evdev.go](/home/admin/gooey/pkg/input/evdev.go):

1. scale raw X and raw Y into screen-space values
2. apply `invertX` and `invertY`
3. if `isLandscape` is `true`, swap X and Y

## `gpio`

The `gpio` section configures GPIO-backed hardware inputs. Today, Gooey supports momentary push-buttons only.

Current built-in default:

```json
"gpio": {
  "buttons": []
}
```

### GPIO Fields

| Field | Type | Required | Default | Notes |
| --- | --- | --- | --- | --- |
| `buttons` | array | No | empty array | List of GPIO-backed momentary buttons |

## `gpio.buttons[]`

Each entry describes one logical button.

Example:

```json
{
  "id": "button1",
  "pin": 23,
  "pull": "up",
  "invert": true,
  "debounceMs": 25,
  "edges": ["press", "release"]
}
```

### Button Fields

| Field | Type | Required | Valid Values | Default | Notes |
| --- | --- | --- | --- | --- | --- |
| `id` | string | Yes | any non-empty string | none | Stable logical ID used by scene `inputBindings` |
| `pin` | string or integer | Yes | integer GPIO number or any pin name accepted by `periph` `gpioreg.ByName()` | none | Stored internally as a string |
| `pull` | string | No | `"float"`, `"up"`, `"down"`, or omitted | omitted, which currently behaves like `float` | Internal pull resistor mode |
| `invert` | boolean | No | `true` or `false` | `false` | Inverts the electrical level to pressed-state mapping |
| `debounceMs` | integer | No | any JSON integer | `0` | Converted to milliseconds |
| `edges` | array of strings | No | any combination of `"press"` and `"release"` | omitted means both | Controls which button phases emit events |

### `pin`

`pin` accepts either:

- a JSON number, for example `23`
- a JSON string, for example `"23"`, `"GPIO23"`, or another name accepted by `periph` on the current board

Current behavior:

- numbers are converted to strings internally
- strings are trimmed of surrounding whitespace
- `null` becomes an empty pin value and will later fail listener initialization

### `pull`

Valid values implemented today:

- `"float"`
- `"up"`
- `"down"`
- omitted or empty string

Current behavior:

- omitted or empty string currently maps to the same runtime behavior as `"float"`
- any other string causes GPIO listener initialization to fail, Gooey logs a warning, and GPIO support is skipped

### `invert`

Current behavior:

- `false`: electrical high means pressed
- `true`: electrical low means pressed

This is useful for buttons wired as active-low with pull-ups.

### `debounceMs`

Current behavior:

- `0` disables software debounce
- positive values debounce successive state changes for the configured duration
- negative values cause GPIO listener creation to fail, Gooey logs a warning, and GPIO support is skipped

### `edges`

Valid values implemented today:

- `"press"`
- `"release"`

Valid examples:

```json
"edges": ["press"]
```

```json
"edges": ["release"]
```

```json
"edges": ["press", "release"]
```

Current behavior:

- omitted or empty array means both `press` and `release`
- order does not matter
- any unsupported value causes GPIO spec parsing to fail and Gooey skips GPIO support
- duplicate values are not explicitly rejected, but provide no extra meaning

## Runtime Failure Behavior

Current Gooey behavior is intentionally tolerant:

- invalid JSON prevents config loading and startup fails
- a bad touch device path logs a warning and Gooey continues without touch
- invalid GPIO button configuration logs a warning and Gooey continues without GPIO input
- if GPIO host initialization or pin lookup fails, Gooey logs a warning and continues without GPIO input

## Relationship To Scene Input Bindings

The `gpio` section only defines physical inputs.

It does not define what those buttons mean on a given screen.

Scene-specific semantics are defined separately in scene `inputBindings`. For example, a GPIO button with `id: "button1"` can mean `next`, `save`, or `cancel` depending on the current scene submitted by the host.

See [docs/scenegraph-reference.md](/home/admin/gooey/docs/scenegraph-reference.md) and [docs/protocol-v1alpha1.md](/home/admin/gooey/docs/protocol-v1alpha1.md) for the host-side scene contract.

## Minimal Examples

### Built-In Defaults Only

```json
{}
```

### Touch Only

```json
{
  "touch": {
    "devicePath": "/dev/input/event0",
    "minXRaw": 200,
    "maxXRaw": 3800,
    "minYRaw": 150,
    "maxYRaw": 3700,
    "ScreenXPixels": 240,
    "ScreenYPixels": 320,
    "invertX": false,
    "invertY": true,
    "isLandscape": true
  }
}
```

### GPIO Buttons With Active-Low Wiring

```json
{
  "gpio": {
    "buttons": [
      {
        "id": "button1",
        "pin": 23,
        "pull": "up",
        "invert": true,
        "debounceMs": 25,
        "edges": ["press", "release"]
      },
      {
        "id": "button2",
        "pin": 22,
        "pull": "up",
        "invert": true,
        "edges": ["press"]
      }
    ]
  }
}
```

## Current Limitations

- No JSON schema file is generated today.
- No config-time validation currently enforces touch min/max consistency.
- No config-time validation currently enforces non-negative component style defaults.
- GPIO currently supports buttons only.
- The `gpio` section does not yet cover encoders, sliders, switches, or LED outputs.

## Related Documents

- [docs/scenegraph-reference.md](/home/admin/gooey/docs/scenegraph-reference.md)
- [docs/protocol-v1alpha1.md](/home/admin/gooey/docs/protocol-v1alpha1.md)
- [docs/host-integration-workflow.md](/home/admin/gooey/docs/host-integration-workflow.md)
- [docs/hardware-smoke-test.md](/home/admin/gooey/docs/hardware-smoke-test.md)
