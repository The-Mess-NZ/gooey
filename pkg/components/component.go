package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

// Component represents a generic graphical UI element in GUIPunk.
type Component interface {
	// ID returns the unique identifier for this component, used for state patching.
	ID() string

	// Draw renders the component onto the provided graphic context.
	Draw(gc *draw2dimg.GraphicContext)

	// BoundingBox returns the physical screen area this component occupies.
	BoundingBox() image.Rectangle

	// HandleTouch processes a touch event mapped to this component.
	// Returns true if the touch resulted in a visual state change requiring a redraw.
	HandleTouch(x, y int, isRelease bool) bool
}
