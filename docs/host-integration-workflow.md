# Gooey Host Integration Workflow

This document describes the recommended host-side workflow for driving Gooey from another process such as Midipunk.

## Ownership Split

Recommended model:

- The host owns the authoritative scenegraph and application state.
- Gooey owns only transient interaction state such as button pressed visual feedback.
- The host converts domain state changes into scene submissions or component patches.
- The host consumes Gooey events and decides what application behavior to trigger.

## Transport

Gooey speaks newline-delimited JSON over a Unix domain socket.

Default development socket:

- `/tmp/guipunk.sock`

Message envelopes are defined in [pkg/ipc/socket.go](/home/admin/gooey/pkg/ipc/socket.go) and protocol constants in [pkg/ipc/protocol.go](/home/admin/gooey/pkg/ipc/protocol.go).

## Recommended Host Lifecycle

1. Start Gooey.
2. Connect one client over the Unix domain socket.
3. Submit a full scene document with `submit_scene`.
4. Read events in a loop.
5. Apply targeted `patch_component` updates when application state changes.
6. Resubmit a full scene when the overall screen or document structure changes substantially.
7. On socket disconnect, reconnect, resubmit the last full scene, and request a status snapshot.

## Command Flow

Initial scene submit:

```json
{
  "action": "submit_scene",
  "payload": {
    "version": "v1alpha1",
    "root": {
      "id": "root",
      "type": "container"
    }
  }
}
```

Targeted patch:

```json
{
  "action": "patch_component",
  "payload": {
    "id": "status",
    "text": "Connected"
  }
}
```

## Event Flow

Typical event sequence for a button tap:

1. `touch_press`
2. `touch_release`
3. `component_activate`

The host should generally treat `component_activate` as the business-level signal and use raw press or release events only when needed for richer interaction handling.

## Go Helper

Gooey now ships a small host-side client helper in [pkg/ipc/client.go](/home/admin/gooey/pkg/ipc/client.go).

Useful reconnect helpers:

- `Reconnect()`
- `ReconnectAndResubmit()`
- `RequestStatus()`
- `RememberedScene()`

Hello World Client Example:

```go
package main

import (
	"log"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
	"github.com/The-Mess-NZ/gui-punk/pkg/ipc"
)

func main() {
	client, err := ipc.Dial("/tmp/guipunk.sock")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	scene := components.SceneDocument{
		Version: components.SceneVersionAlpha1,
		Root: components.SceneNode{
			ID:   "root",
			Type: components.NodeTypeContainer,
			Children: []components.SceneNode{
				{ID: "title", Type: components.NodeTypeLabel, Text: "Hello World"},
			},
		},
	}

	if err := client.SubmitScene(scene); err != nil {
		log.Fatal(err)
	}

	for {
		event, err := client.NextEvent()
		if err != nil {
			log.Fatal(err)
		}

		switch event.Type {
		case ipc.EventComponentActivate:
			var interaction components.Interaction
			if err := ipc.DecodeEventPayload(event, &interaction); err != nil {
				log.Fatal(err)
			}
			log.Printf("action=%s component=%s", interaction.Action, interaction.ComponentID)
		}
	}
}
```

## Recommended Update Strategy

Use `submit_scene` when:

- You are showing a new screen.
- The document structure changes significantly.
- Host and renderer state may have drifted and you want a full reset.

Use `patch_component` when:

- A label or button caption changes.
- A visual state indicator changes.
- Visibility changes for a single node.
- Bounds or style of a known node change without restructuring the screen.

## Error Handling

Watch for:

- `scene_loaded`
- `component_patched`
- `protocol_error`

Structured `protocol_error` payloads now include a stable `code` field in addition to the human-readable `message`.

If `protocol_error` arrives, log it, inspect the rejected command, and prefer resubmitting a full scene when recovery is ambiguous.

After reconnect, request `get_status` and inspect the `diagnostics` payload to confirm that Gooey is running, whether a scene is currently loaded, and which root document Gooey believes is active.

## Midipunk Mapping Pattern

Recommended pattern for Midipunk:

- Translate config and router state into Gooey scene nodes in Midipunk.
- Map route or port IDs directly into stable scene node IDs.
- Use Gooey `action` strings for UI intent such as `toggle_route` or `select_port`.
- Convert `component_activate` events back into Midipunk domain commands.

This keeps Gooey generic while letting Midipunk stay the source of truth.

## References

- [docs/protocol-v1alpha1.md](/home/admin/gooey/docs/protocol-v1alpha1.md)
- [docs/scenegraph-reference.md](/home/admin/gooey/docs/scenegraph-reference.md)
- [docs/examples/alpha-demo-scene.json](/home/admin/gooey/docs/examples/alpha-demo-scene.json)
- [docs/hardware-smoke-test.md](/home/admin/gooey/docs/hardware-smoke-test.md)
