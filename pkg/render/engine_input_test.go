package render

import (
	"testing"

	"github.com/The-Mess-NZ/gooey/pkg/components"
	"github.com/The-Mess-NZ/gooey/pkg/input"
)

func TestEngineHandleInputEventEmitsMappedReport(t *testing.T) {
	engine := &Engine{
		scene: &components.SceneDocument{
			InputBindings: []components.InputBinding{{ID: "next", OnPress: "next_item", OnRelease: "back_item"}},
		},
	}

	var gotType string
	var got input.Report
	engine.SetEventHandler(func(eventType string, payload any) {
		report, ok := payload.(input.Report)
		if !ok {
			t.Fatalf("payload type = %T, want input.Report", payload)
		}
		gotType = eventType
		got = report
	})

	engine.HandleInputEvent(input.Event{
		Source:      input.SourceGPIO,
		InputID:     "next",
		ControlType: input.ControlTypeButton,
		Phase:       input.PhasePress,
	})

	if gotType != "input_event" {
		t.Fatalf("event type = %q, want %q", gotType, "input_event")
	}
	if got.Action != "next_item" {
		t.Fatalf("action = %q, want %q", got.Action, "next_item")
	}
	if got.Phase != input.PhasePress {
		t.Fatalf("phase = %q, want %q", got.Phase, input.PhasePress)
	}
}

func TestEngineHandleInputEventIgnoresUnboundPhase(t *testing.T) {
	engine := &Engine{
		scene: &components.SceneDocument{
			InputBindings: []components.InputBinding{{ID: "next", OnPress: "next_item"}},
		},
	}

	called := false
	engine.SetEventHandler(func(_ string, _ any) {
		called = true
	})

	engine.HandleInputEvent(input.Event{
		Source:      input.SourceGPIO,
		InputID:     "next",
		ControlType: input.ControlTypeButton,
		Phase:       input.PhaseRelease,
	})

	if called {
		t.Fatal("HandleInputEvent() emitted an event for an unmapped phase")
	}
}
