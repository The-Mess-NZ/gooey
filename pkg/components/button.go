package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

type buttonComponent struct {
	id      string
	bbox    image.Rectangle
	text    string
	action  string
	style   Style
	pressed bool
}

func init() {
	RegisterComponent(NodeTypeButton, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		return &buttonComponent{id: node.ID, bbox: rect, text: node.Text, action: node.Action, style: style}, nil
	})
}

func (c *buttonComponent) ID() string {
	return c.id
}

func (c *buttonComponent) Draw(gc *draw2dimg.GraphicContext) {
	style := c.style
	if c.pressed {
		style.Background = darkenColor(style.Background, 0.25)
	}
	drawBox(gc, c.bbox, style)
	drawText(gc, c.bbox, c.text, style, true)
}

func (c *buttonComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *buttonComponent) HandleTouch(x, y int, isRelease bool) TouchResult {
	inside := image.Pt(x, y).In(c.bbox)
	if !isRelease {
		if !inside || c.pressed {
			return TouchResult{}
		}
		c.pressed = true
		return TouchResult{NeedsRedraw: true}
	}

	if !c.pressed {
		return TouchResult{}
	}

	c.pressed = false
	result := TouchResult{NeedsRedraw: true}
	if inside {
		result.Interactions = append(result.Interactions, Interaction{
			Kind:        "component_activate",
			ComponentID: c.id,
			Action:      c.action,
			X:           x,
			Y:           y,
			LocalX:      x - c.bbox.Min.X,
			LocalY:      y - c.bbox.Min.Y,
		})
	}
	return result
}
