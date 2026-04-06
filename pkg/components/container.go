package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

type containerComponent struct {
	id      string
	bbox    image.Rectangle
	style   Style
	action  string
	pressed bool
}

func init() {
	RegisterComponent(NodeTypeContainer, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		return &containerComponent{id: node.ID, bbox: rect, style: style, action: node.Action}, nil
	})
}

func (c *containerComponent) ID() string {
	return c.id
}

func (c *containerComponent) Draw(gc *draw2dimg.GraphicContext) {
	style := c.style
	if c.pressed && c.action != "" {
		style.Background = darkenColor(style.Background, 0.2)
	}
	drawBox(gc, c.bbox, style)
}

func (c *containerComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *containerComponent) HandleTouch(x, y int, isRelease bool) TouchResult {
	if c.action == "" {
		return TouchResult{}
	}

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
