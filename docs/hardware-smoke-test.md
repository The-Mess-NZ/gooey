# Gooey Hardware Smoke Test

This checklist is the minimum manual validation pass for a Gooey alpha build on target hardware.

## Preconditions

- The framebuffer device is available.
- The touch device path in [config.json](/home/admin/gooey/config.json) is correct.
- Gooey builds successfully with `go build ./...`.
- A known test scene is available, such as [docs/examples/alpha-demo-scene.json](/home/admin/gooey/docs/examples/alpha-demo-scene.json).

## 1. Boot And Render

1. Start Gooey.
2. Start a host client or the demo client.
3. Confirm the screen is not blank.
4. Confirm the root background, labels, and buttons are visible and remain inside screen bounds.

Expected result:

- No clipped root layout.
- No text rendering outside node bounds.
- No crash on initial scene submission.

## 2. Scene Submit And Patch

1. Submit a full scene with `submit_scene`.
2. Update at least one label with `patch_component`.
3. Update at least one style value with `patch_component`.

Expected result:

- `scene_loaded` arrives after the initial scene submit.
- `component_patched` arrives after targeted updates.
- The screen changes without requiring a Gooey restart.

## 3. Touch Calibration

1. Run the calibration client.
2. Complete the four-corner calibration sequence.
3. Confirm [config.json](/home/admin/gooey/config.json) was updated.
4. Restart Gooey.

Expected result:

- Touch configuration persists across restart.
- The saved device path and raw min or max bounds match the calibrated hardware.

## 4. Touch Interaction

1. Tap a button in the demo scene.
2. Tap near the edges of multiple components.
3. Tap an empty area.

Expected result:

- `touch_press` and `touch_release` are emitted.
- `component_activate` is emitted only when the release occurs inside an interactive component.
- Empty-area taps do not activate components.

## 5. GPIO Input Interaction

1. Configure at least one GPIO button in [config.json](/home/admin/gooey/config.json).
2. Submit a scene with `inputBindings` for that button.
3. Press and release the physical button.
4. Repeat with a debounce-sensitive rapid tap sequence.

Expected result:

- Gooey starts even if GPIO is omitted from config.
- `input_event` is emitted only for configured and scene-bound GPIO inputs.
- `phase` reflects whether the button press or release triggered the event.
- Debounce suppresses switch chatter without preventing normal presses and releases.

## 6. Protocol Errors

1. Send malformed JSON over the socket.
2. Send a command with no `action`.
3. Send a scene with duplicate IDs or an unsupported component type.
4. Send a patch with no `id`.

Expected result:

- Gooey stays running.
- `protocol_error` events are emitted with a useful `code` and `message`.
- Valid later commands still succeed.

## 7. Reconnect Behavior

1. Connect a host client and submit a scene.
2. Disconnect the client.
3. Reconnect a client.
4. Resubmit the scene.
5. Request a status snapshot with `get_status`.

Expected result:

- Gooey remains running across disconnects.
- The reconnecting host can resubmit and continue driving the UI.
- A `diagnostics` event returns a coherent runtime status snapshot after reconnect, including GPIO status when configured.
- No panic or stale socket failure occurs.

## 8. Diagnostics To Capture

When a test fails, capture:

- Gooey stdout or stderr logs.
- The exact `protocol_error` payload.
- The current [config.json](/home/admin/gooey/config.json).
- The exact scene or patch payload that triggered the problem.
- The raw evdev behavior if touch calibration or touch delivery is wrong.
