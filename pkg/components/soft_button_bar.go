package components

import (
	"image"
	"image/color"

	"github.com/llgcode/draw2d/draw2dimg"
)

type softButtonBarComponent struct {
	id    string
	bbox  image.Rectangle
	style Style
	slots [4]string
}

func init() {
	RegisterComponent(NodeTypeSoftBar, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		component := &softButtonBarComponent{id: node.ID, bbox: rect, style: style}
		for index := 0; index < len(node.SoftButtons) && index < len(component.slots); index++ {
			component.slots[index] = node.SoftButtons[index].Label
		}
		return component, nil
	})
}

func (c *softButtonBarComponent) ID() string {
	return c.id
}

func (c *softButtonBarComponent) Draw(gc *draw2dimg.GraphicContext) {
	if c.bbox.Empty() {
		return
	}

	drawBox(gc, c.bbox, c.style)

	slots := splitRectEvenly(c.bbox, len(c.slots))
	divider, ok := parseHexColor(c.style.BorderColor)
	if !ok {
		divider = c.styleColorFallback()
	}
	gc.SetStrokeColor(divider)
	gc.SetLineWidth(float64(max(1, c.style.BorderWidth)))
	for index, rect := range slots {
		if index > 0 {
			x := float64(rect.Min.X)
			gc.BeginPath()
			gc.MoveTo(x, float64(c.bbox.Min.Y))
			gc.LineTo(x, float64(c.bbox.Max.Y))
			gc.Stroke()
		}
		drawText(gc, rect, c.slots[index], c.style, true)
	}
}

func (c *softButtonBarComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *softButtonBarComponent) HandleTouch(_, _ int, _ bool) TouchResult {
	return TouchResult{}
}

func (c *softButtonBarComponent) styleColorFallback() color.RGBA {
	foreground, ok := parseHexColor(c.style.Foreground)
	if ok {
		return foreground
	}
	return color.RGBA{168, 194, 212, 255}
}

// TODO: This could be extracted into a wider gui helper library or something - e.g. drawing.go
func splitRectEvenly(rect image.Rectangle, count int) []image.Rectangle {
	if count <= 0 || rect.Empty() {
		return nil
	}

	width := rect.Dx() / count
	remainder := rect.Dx() % count
	parts := make([]image.Rectangle, count)
	cursor := rect.Min.X
	for index := range count {
		partWidth := width
		if remainder > 0 {
			partWidth++
			remainder--
		}
		parts[index] = image.Rect(cursor, rect.Min.Y, cursor+partWidth, rect.Max.Y)
		cursor += partWidth
	}
	return parts
}
