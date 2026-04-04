package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

type containerComponent struct {
	id    string
	bbox  image.Rectangle
	style Style
}

func init() {
	RegisterComponent(NodeTypeContainer, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		return &containerComponent{id: node.ID, bbox: rect, style: style}, nil
	})
}

func (c *containerComponent) ID() string {
	return c.id
}

func (c *containerComponent) Draw(gc *draw2dimg.GraphicContext) {
	drawBox(gc, c.bbox, c.style)
}

func (c *containerComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *containerComponent) HandleTouch(x, y int, isRelease bool) TouchResult {
	return TouchResult{}
}
