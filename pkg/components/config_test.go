package components

import (
	"encoding/json"
	"testing"
)

func TestGPIOPinUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		data string
		want GPIOPin
	}{
		{name: "string", data: `"GPIO17"`, want: GPIOPin("GPIO17")},
		{name: "number", data: `17`, want: GPIOPin("17")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var pin GPIOPin
			if err := json.Unmarshal([]byte(test.data), &pin); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if pin != test.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", pin, test.want)
			}
		})
	}
}

func TestMergeGPIOConfigClonesOverrideButtons(t *testing.T) {
	base := GPIOConfig{Buttons: []GPIOButtonConfig{{ID: "base", Pin: GPIOPin("17"), Edges: []string{GPIOEdgePress}}}}
	override := GPIOConfig{Buttons: []GPIOButtonConfig{{ID: "next", Pin: GPIOPin("27"), Edges: []string{GPIOEdgeRelease}}}}

	merged := mergeGPIOConfig(base, override)
	override.Buttons[0].Edges[0] = "mutated"

	if len(merged.Buttons) != 1 {
		t.Fatalf("mergeGPIOConfig() button count = %d, want 1", len(merged.Buttons))
	}
	if merged.Buttons[0].ID != "next" {
		t.Fatalf("mergeGPIOConfig() id = %q, want %q", merged.Buttons[0].ID, "next")
	}
	if merged.Buttons[0].Edges[0] != GPIOEdgeRelease {
		t.Fatalf("mergeGPIOConfig() retained shared edge slice = %q", merged.Buttons[0].Edges[0])
	}
}
