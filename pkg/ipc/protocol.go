package ipc

const (
	ActionSubmitScene    = "submit_scene"
	ActionReplaceScene   = "replace_scene"
	ActionPatchComponent = "patch_component"

	EventTouchPress        = "touch_press"
	EventTouchRelease      = "touch_release"
	EventComponentActivate = "component_activate"
	EventSceneLoaded       = "scene_loaded"
	EventComponentPatched  = "component_patched"
	EventProtocolError     = "protocol_error"
)

// AckPayload confirms successful command handling.
type AckPayload struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

// ErrorPayload reports a rejected or malformed command.
type ErrorPayload struct {
	Action  string `json:"action,omitempty"`
	Message string `json:"message"`
}
