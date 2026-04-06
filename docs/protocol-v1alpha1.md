# Gooey Protocol v1alpha1

Gooey exposes a newline-delimited JSON protocol over a Unix domain socket. The host application owns the authoritative scenegraph and submits either a full scene document or targeted component patches.

This file is the transport-level protocol summary. For the accepted scene schema, see [docs/scenegraph-reference.md](/home/admin/gooey/docs/scenegraph-reference.md). For the recommended host-side usage pattern, see [docs/host-integration-workflow.md](/home/admin/gooey/docs/host-integration-workflow.md).

## Commands

Each command uses the existing transport envelope:

```json
{"action":"submit_scene","payload":{...}}
```

Supported `action` values in this implementation:

- `submit_scene`: Load a scene document and replace any current scene.
- `replace_scene`: Alias of `submit_scene` for protocol clarity.
- `patch_component`: Apply a targeted patch to a node identified by `id`.
- `get_status`: Request a runtime status snapshot from Gooey.

## Scene Document

The payload for `submit_scene` and `replace_scene` is a `SceneDocument`. A full field-by-field reference lives in [docs/scenegraph-reference.md](/home/admin/gooey/docs/scenegraph-reference.md).

```json
{
  "version": "v1alpha1",
  "inputBindings": [
    {
      "id": "next",
      "onPress": "next_item"
    },
    {
      "id": "back",
      "onRelease": "cancel"
    }
  ],
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

Supported built-in node types in this implementation:

- `container`
- `label`
- `button`
- `toggle`
- `soft_button_bar`

Supported automatic layout directions:

- `vertical`
- `horizontal`

## Component Patch

The payload for `patch_component` is a `ComponentPatch` that targets one node by ID.

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
- `checked`
- `visible`

`soft_button_bar` nodes carry a four-slot `softButtons` payload in the submitted scene document:

```json
{
  "id": "navigation-bar",
  "type": "soft_button_bar",
  "bounds": { "height": 34 },
  "softButtons": [
    { "label": "<<" },
    { "label": "<" },
    { "label": ">" },
    { "label": ">>" }
  ]
}
```

Each bar must define exactly four slots. Empty labels still render blank boxes so the host can show unused hardware button positions without assigning them an action.

## Events

Gooey emits newline-delimited JSON events using the transport envelope:

```json
{"type":"touch_press","payload":{...}}
```

Events currently emitted:

- `touch_press`
- `touch_release`
- `component_activate`
- `input_event`
- `scene_loaded`
- `component_patched`
- `protocol_error`
- `diagnostics`

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

`input_event` is emitted for scene-bound hardware inputs such as GPIO buttons. Example:

```json
{
  "source": "gpio",
  "inputId": "next",
  "controlType": "button",
  "phase": "press",
  "action": "next_item"
}
```

`diagnostics` currently carries either connection lifecycle notices or a status snapshot.

Status snapshot example:

```json
{
  "kind": "status_snapshot",
  "message": "current Gooey runtime status",
  "status": {
    "sceneLoaded": true,
    "sceneVersion": "v1alpha1",
    "rootId": "root",
    "componentCount": 7,
    "touchConfigured": true,
    "touchDevicePath": "/dev/input/event1",
    "gpioConfigured": true,
    "gpioInputCount": 2,
    "configPath": "./config.json"
  }
}
```

## Error Model

`protocol_error` payloads now include:

- `action`: The command action when available.
- `code`: A stable machine-readable failure code.
- `message`: A human-readable description.

Current codes:

- `invalid_envelope`
- `invalid_command`
- `invalid_scene`
- `invalid_patch`
- `scene_not_loaded`
- `unknown_action`
- `internal_error`

## Host Helper

For Go hosts, a small reusable client helper is available in [pkg/ipc/client.go](/home/admin/gooey/pkg/ipc/client.go).
