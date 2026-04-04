package components

import (
	"image"

	"github.com/llgcode/draw2d/draw2dimg"
)

type labelComponent struct {
	id    string
	bbox  image.Rectangle
	text  string
	style Style
}

func init() {
	RegisterComponent(NodeTypeLabel, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		return &labelComponent{id: node.ID, bbox: rect, text: node.Text, style: style}, nil
	})
}

func (c *labelComponent) ID() string {
	return c.id
}

func (c *labelComponent) Draw(gc *draw2dimg.GraphicContext) {
	drawBox(gc, c.bbox, c.style)
	drawText(gc, c.bbox, c.text, c.style, false)
}

func (c *labelComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *labelComponent) HandleTouch(x, y int, isRelease bool) TouchResult {
	return TouchResult{}
}
