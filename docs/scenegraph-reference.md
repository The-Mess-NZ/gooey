# Gooey Scenegraph Reference

This document describes the scenegraph schema currently accepted by Gooey `v1alpha1`.

## Model

The host application owns the authoritative scenegraph and submits it to Gooey over the Unix domain socket protocol. Gooey renders that scene, tracks transient widget interaction state, and emits interaction events back to the host.

Top-level document shape:

```json
{
  "version": "v1alpha1",
  "inputBindings": [
    {
      "id": "next",
      "onPress": "next_item"
    }
  ],
  "root": {
    "id": "root",
    "type": "container"
  }
}
```

## Root Document

Fields:

- `version`: Optional. Defaults to `v1alpha1` when omitted.
- `inputBindings`: Optional. Scene-specific bindings for stable hardware input IDs such as GPIO buttons.
- `root`: Required. A single `SceneNode` that becomes the root of the rendered tree.

## InputBinding

Fields:

- `id`: Required. Stable hardware input identifier that matches Gooey runtime config.
- `controlType`: Optional. Defaults to `button`. Other control types are reserved for future protocol revisions.
- `onPress`: Optional. Host-defined semantic action emitted when a button is pressed.
- `onRelease`: Optional. Host-defined semantic action emitted when a button is released.

At least one of `onPress` or `onRelease` must be set for each binding.

## SceneNode

Fields:

- `id`: Required. Must be unique within the submitted scene.
- `type`: Required. Must be one of the registered component types.
- `bounds`: Optional. Position or requested size for the node.
- `layout`: Optional. Container layout settings for children.
- `style`: Optional. Visual styling for the node.
- `text`: Optional. Used by text-capable components such as `label` and `button`.
- `action`: Optional. Host-defined action string emitted by interactive components.
- `checked`: Optional. Boolean state used by `toggle` components.
- `visible`: Optional. Defaults to `true` when omitted.
- `children`: Optional. Nested child nodes.

## Supported Built-in Component Types

Current built-ins:

- `container`: Draws an optional background or border and lays out child nodes.
- `label`: Draws wrapped or truncated text within its bounds.
- `button`: Draws a pressable box with centered text and emits `component_activate` on release inside bounds.
- `toggle`: Draws a switch control based on `checked` and emits `component_activate` on release inside bounds.

Built-ins are registered in separate files under [pkg/components/container.go](/home/admin/gooey/pkg/components/container.go), [pkg/components/label.go](/home/admin/gooey/pkg/components/label.go), and [pkg/components/button.go](/home/admin/gooey/pkg/components/button.go). Custom component types can be added by registering another factory with `components.RegisterComponent`.

## Bounds

JSON shape:

```json
{
  "x": 0,
  "y": 0,
  "width": 320,
  "height": 40
}
```

Rules:

- `x` and `y` are offsets relative to the parent content box when `bounds` is interpreted directly.
- In layout containers, `width` or `height` may be omitted to request automatic sizing on the layout axis.
- Final layout rectangles are clamped to the parent container bounds.

## Layout

JSON shape:

```json
{
  "direction": "vertical",
  "gap": 8,
  "padding": {
    "top": 8,
    "right": 8,
    "bottom": 8,
    "left": 8
  }
}
```

Supported directions:

- `vertical`
- `horizontal`

Behavior:

- If `direction` is omitted, children use explicit `bounds` when present, otherwise they inherit the full parent content box.
- In directional layouts, explicit width or height values reserve fixed space on the layout axis.
- Remaining space is divided evenly across children without an explicit size on the layout axis.
- `gap` is applied between siblings.
- `padding` reduces the usable content box inside the container.

## Style

JSON shape:

```json
{
  "background": "#1B2732FF",
  "foreground": "#F7F1D5",
  "borderColor": "#9ED8B5",
  "borderWidth": 2,
  "fontSize": 14,
  "textPadding": 4
}
```

Supported fields:

- `background`: Optional RGBA or RGB hex string.
- `foreground`: Optional RGBA or RGB hex string for text.
- `borderColor`: Optional RGBA or RGB hex string.
- `borderWidth`: Optional border stroke width in pixels.
- `fontSize`: Optional text size override.
- `textPadding`: Optional inner padding used by text layout.

Defaults are loaded from [config.json](/home/admin/gooey/config.json). See [docs/config-reference.md](/home/admin/gooey/docs/config-reference.md) for the full Gooey runtime config reference.

## Patch Model

Hosts can update one node by ID using `patch_component`.

Supported patch fields:

- `bounds`
- `style`
- `text`
- `action`
- `checked`
- `visible`

Patch example:

```json
{
  "id": "status",
  "text": "Route enabled",
  "style": {
    "background": "#214357FF",
    "foreground": "#F2F2E9"
  }
}
```

Patches replace the provided field values on the target node, then Gooey rebuilds the renderable component tree.

## Example Scene

See [docs/examples/alpha-demo-scene.json](/home/admin/gooey/docs/examples/alpha-demo-scene.json) for a complete working example that matches the current demo client.

## Constraints

- IDs must be unique per scene.
- Input binding IDs must be unique per scene.
- Unsupported component types are rejected.
- Gooey is currently single-client over the UDS transport.
- Host applications remain the source of truth for scene and business state.
