package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

// Interaction describes a host-facing touch or component event.
type Interaction struct {
	Kind        string `json:"kind"`
	ComponentID string `json:"componentId,omitempty"`
	Action      string `json:"action,omitempty"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	LocalX      int    `json:"localX,omitempty"`
	LocalY      int    `json:"localY,omitempty"`
}

// TouchResult captures the result of a touch interaction with a component.
type TouchResult struct {
	NeedsRedraw  bool
	Interactions []Interaction
}

// Component represents a generic graphical UI element in GUIPunk.
type Component interface {
	// ID returns the unique identifier for this component, used for state patching.
	ID() string

	// Draw renders the component onto the provided graphic context.
	Draw(gc *draw2dimg.GraphicContext)

	// BoundingBox returns the physical screen area this component occupies.
	BoundingBox() image.Rectangle

	// HandleTouch processes a touch event mapped to this component.
	// It returns any redraw requirement and optional host-facing component events.
	HandleTouch(x, y int, isRelease bool) TouchResult
}
