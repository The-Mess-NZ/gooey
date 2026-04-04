package components

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

type containerComponent struct {
	id    string
	bbox  image.Rectangle
	style Style
}

type labelComponent struct {
	id    string
	bbox  image.Rectangle
	text  string
	style Style
}

type buttonComponent struct {
	id      string
	bbox    image.Rectangle
	text    string
	action  string
	style   Style
	pressed bool
}

func newContainerComponent(id string, bbox image.Rectangle, style Style) Component {
	return &containerComponent{id: id, bbox: bbox, style: style}
}

func newLabelComponent(id string, bbox image.Rectangle, text string, style Style) Component {
	return &labelComponent{id: id, bbox: bbox, text: text, style: style}
}

func newButtonComponent(id string, bbox image.Rectangle, text, action string, style Style) Component {
	return &buttonComponent{id: id, bbox: bbox, text: text, action: action, style: style}
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

func defaultStyle(nodeType string) Style {
	switch nodeType {
	case NodeTypeButton:
		return Style{
			Background:  "#2F6B4F",
			Foreground:  "#F2F2E9",
			BorderColor: "#9ED8B5",
			BorderWidth: 2,
			FontSize:    18,
		}
	case NodeTypeLabel:
		return Style{
			Foreground: "#F2F2E9",
			FontSize:   16,
		}
	default:
		return Style{}
	}
}

func mergeStyle(nodeType string, override *Style) Style {
	style := defaultStyle(nodeType)
	if override == nil {
		return style
	}
	if override.Background != "" {
		style.Background = override.Background
	}
	if override.Foreground != "" {
		style.Foreground = override.Foreground
	}
	if override.BorderColor != "" {
		style.BorderColor = override.BorderColor
	}
	if override.BorderWidth != 0 {
		style.BorderWidth = override.BorderWidth
	}
	if override.FontSize != 0 {
		style.FontSize = override.FontSize
	}
	return style
}

func drawBox(gc *draw2dimg.GraphicContext, bbox image.Rectangle, style Style) {
	if bbox.Empty() {
		return
	}

	fill, hasFill := parseHexColor(style.Background)
	if !hasFill {
		fill = color.RGBA{0, 0, 0, 0}
	}

	border, hasBorder := parseHexColor(style.BorderColor)
	if !hasBorder {
		border = fill
	}

	gc.SetFillColor(fill)
	gc.SetStrokeColor(border)
	gc.SetLineWidth(float64(style.BorderWidth))
	draw2dkit.Rectangle(gc, float64(bbox.Min.X), float64(bbox.Min.Y), float64(bbox.Max.X), float64(bbox.Max.Y))
	if style.BorderWidth > 0 {
		gc.FillStroke()
		return
	}
	gc.Fill()
}

func drawText(gc *draw2dimg.GraphicContext, bbox image.Rectangle, text string, style Style, centered bool) {
	if text == "" || bbox.Empty() {
		return
	}

	fg, ok := parseHexColor(style.Foreground)
	if !ok {
		fg = color.RGBA{255, 255, 255, 255}
	}
	fontSize := style.FontSize
	if fontSize == 0 {
		fontSize = 16
	}

	gc.SetFillColor(fg)
	gc.SetFontSize(fontSize)

	x := float64(bbox.Min.X + 8)
	if centered {
		approxCharWidth := fontSize * 0.55
		textWidth := approxCharWidth * float64(len(text))
		x = float64(bbox.Min.X) + float64(bbox.Dx())/2 - textWidth/2
		if x < float64(bbox.Min.X+6) {
			x = float64(bbox.Min.X + 6)
		}
	}
	y := float64(bbox.Min.Y) + float64(bbox.Dy())*0.62
	gc.FillStringAt(text, x, y)
}

func darkenColor(hex string, amount float64) string {
	parsed, ok := parseHexColor(hex)
	if !ok {
		parsed = color.RGBA{47, 107, 79, 255}
	}
	shade := func(v uint8) uint8 {
		value := float64(v) * (1 - amount)
		if value < 0 {
			value = 0
		}
		return uint8(value)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", shade(parsed.R), shade(parsed.G), shade(parsed.B), parsed.A)
}

func parseHexColor(value string) (color.RGBA, bool) {
	if value == "" {
		return color.RGBA{}, false
	}

	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(trimmed) != 6 && len(trimmed) != 8 {
		return color.RGBA{}, false
	}

	parsed, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}

	if len(trimmed) == 6 {
		return color.RGBA{
			R: uint8(parsed >> 16),
			G: uint8((parsed >> 8) & 0xFF),
			B: uint8(parsed & 0xFF),
			A: 0xFF,
		}, true
	}

	return color.RGBA{
		R: uint8(parsed >> 24),
		G: uint8((parsed >> 16) & 0xFF),
		B: uint8((parsed >> 8) & 0xFF),
		A: uint8(parsed & 0xFF),
	}, true
}
