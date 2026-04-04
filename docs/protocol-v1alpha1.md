# Gooey Protocol v1alpha1

Gooey exposes a newline-delimited JSON protocol over a Unix domain socket. The host application owns the authoritative scenegraph and submits either a full scene document or targeted component patches.

## Commands

Each command uses the existing transport envelope:

```json
{"action":"submit_scene","payload":{...}}
```

Supported `action` values in this implementation:

- `submit_scene`: Load a scene document and replace any current scene.
- `replace_scene`: Alias of `submit_scene` for protocol clarity.
- `patch_component`: Apply a targeted patch to a node identified by `id`.

## Scene Document

```json
{
  "version": "v1alpha1",
  "root": {
    "id": "root",
    "type": "container",
    "layout": {
      "direction": "vertical",
      "gap": 8,
      "padding": { "top": 12, "right": 12, "bottom": 12, "left": 12 }
    },
    "style": {
      "background": "#10161BFF"
    },
    "children": [
      {
        "id": "title",
        "type": "label",
        "text": "MidiPunk",
        "bounds": { "height": 40 }
      },
      {
        "id": "route-toggle",
        "type": "button",
        "text": "Enable Route",
        "action": "toggle_route"
      }
    ]
  }
}
```

Supported node types in this implementation:

- `container`
- `label`
- `button`

Supported automatic layout directions:

- `vertical`
- `horizontal`

## Component Patch

```json
{
  "id": "route-toggle",
  "text": "Disable Route",
  "style": {
    "background": "#8B3A3AFF",
    "borderColor": "#F0C6B8FF"
  }
}
```

Supported patch fields in this implementation:

- `bounds`
- `style`
- `text`
- `action`
- `visible`

## Events

Gooey emits newline-delimited JSON events using the transport envelope:

```json
{"type":"touch_press","payload":{...}}
```

Events currently emitted:

- `touch_press`
- `touch_release`
- `component_activate`
- `scene_loaded`
- `component_patched`
- `protocol_error`

`touch_press`, `touch_release`, and `component_activate` share this payload shape:

```json
{
  "kind": "component_activate",
  "componentId": "route-toggle",
  "action": "toggle_route",
  "x": 184,
  "y": 126,
  "localX": 40,
  "localY": 18
}
```

For alpha, Gooey emits press/release plus component activation. High-volume move or gesture streaming is intentionally out of scope.
