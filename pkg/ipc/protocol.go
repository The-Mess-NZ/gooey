package ipc

const (
	ActionSubmitScene    = "submit_scene"
	ActionReplaceScene   = "replace_scene"
	ActionPatchComponent = "patch_component"
	ActionGetStatus      = "get_status"

	EventTouchPress        = "touch_press"
	EventTouchRelease      = "touch_release"
	EventComponentActivate = "component_activate"
	EventSceneLoaded       = "scene_loaded"
	EventComponentPatched  = "component_patched"
	EventProtocolError     = "protocol_error"
	EventDiagnostics       = "diagnostics"
)

const (
	ErrorCodeInvalidEnvelope = "invalid_envelope"
	ErrorCodeInvalidCommand  = "invalid_command"
	ErrorCodeInvalidScene    = "invalid_scene"
	ErrorCodeInvalidPatch    = "invalid_patch"
	ErrorCodeSceneNotLoaded  = "scene_not_loaded"
	ErrorCodeUnknownAction   = "unknown_action"
	ErrorCodeInternal        = "internal_error"
)

// AckPayload confirms successful command handling.
type AckPayload struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

// ErrorPayload reports a rejected or malformed command.
type ErrorPayload struct {
	Action  string `json:"action,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// StatusPayload reports the current Gooey runtime state.
type StatusPayload struct {
	SceneLoaded     bool   `json:"sceneLoaded"`
	SceneVersion    string `json:"sceneVersion,omitempty"`
	RootID          string `json:"rootId,omitempty"`
	ComponentCount  int    `json:"componentCount"`
	TouchConfigured bool   `json:"touchConfigured"`
	TouchDevicePath string `json:"touchDevicePath,omitempty"`
	ConfigPath      string `json:"configPath,omitempty"`
}

// DiagnosticsPayload reports non-fatal runtime details useful for recovery.
type DiagnosticsPayload struct {
	Kind    string         `json:"kind"`
	Message string         `json:"message,omitempty"`
	Status  *StatusPayload `json:"status,omitempty"`
}
