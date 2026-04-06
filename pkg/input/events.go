package input

import (
	"fmt"
	"strings"
	"time"
)

const (
	SourceGPIO = "gpio"

	ControlTypeButton = "button"

	PhasePress   = "press"
	PhaseRelease = "release"
)

// Event is a normalized hardware input event emitted by a device listener.
type Event struct {
	Source      string   `json:"source"`
	InputID     string   `json:"inputId"`
	ControlType string   `json:"controlType"`
	Phase       string   `json:"phase,omitempty"`
	Delta       int      `json:"delta,omitempty"`
	Value       *float64 `json:"value,omitempty"`
}

// Report is the scene-aware hardware input payload sent to the host.
type Report struct {
	Source      string   `json:"source"`
	InputID     string   `json:"inputId"`
	ControlType string   `json:"controlType"`
	Phase       string   `json:"phase,omitempty"`
	Delta       int      `json:"delta,omitempty"`
	Value       *float64 `json:"value,omitempty"`
	Action      string   `json:"action,omitempty"`
}

// EventHandler consumes normalized hardware input events.
type EventHandler interface {
	HandleInputEvent(event Event)
}

// GPIOButtonSpec configures one GPIO-backed button input listener.
type GPIOButtonSpec struct {
	ID          string        // unique identifier for this button, e.g. "volume_up"
	Pin         string        // accepts BCM GPIO pin numbers as strings, e.g. "17" = BCM GPIO17
	Pull        string        // "float", "up", or "down"
	Invert      bool          // whether to invert the raw input state, e.g. for active-low buttons
	Debounce    time.Duration // debounce duration to filter out rapid state changes, e.g. 50ms
	EmitPress   bool          // whether to emit events for press (falling edge) transitions
	EmitRelease bool          // whether to emit events for release (rising edge) transitions
}

// EmitPhases resolves config edge names into press and release emission flags.
func EmitPhases(edges []string) (bool, bool, error) {
	if len(edges) == 0 {
		return true, true, nil
	}

	emitPress := false
	emitRelease := false
	for _, edge := range edges {
		switch strings.ToLower(strings.TrimSpace(edge)) {
		case PhasePress:
			emitPress = true
		case PhaseRelease:
			emitRelease = true
		default:
			return false, false, fmt.Errorf("unsupported gpio edge %q", edge)
		}
	}

	if !emitPress && !emitRelease {
		return false, false, fmt.Errorf("at least one gpio edge must be enabled")
	}
	return emitPress, emitRelease, nil
}
