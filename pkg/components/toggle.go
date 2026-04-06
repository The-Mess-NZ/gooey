package components

import (
	"image"
	"image/color"

	"github.com/llgcode/draw2d/draw2dimg"
)

type toggleComponent struct {
	id      string
	bbox    image.Rectangle
	checked bool
	action  string
	style   Style
	pressed bool
}

func init() {
	RegisterComponent(NodeTypeToggle, func(node SceneNode, rect image.Rectangle, style Style) (Component, error) {
		checked := node.Checked != nil && *node.Checked
		return &toggleComponent{id: node.ID, bbox: rect, checked: checked, action: node.Action, style: style}, nil
	})
}

func (c *toggleComponent) ID() string {
	return c.id
}

func (c *toggleComponent) Draw(gc *draw2dimg.GraphicContext) {
	track := toggleTrackRect(c.bbox)
	if track.Empty() {
		return
	}

	trackStyle := c.style
	if !c.checked {
		trackStyle.Background = "#2D353CFF"
		if trackStyle.BorderColor == "" {
			trackStyle.BorderColor = "#778A97FF"
		}
	} else if c.pressed {
		trackStyle.Background = darkenColor(trackStyle.Background, 0.18)
	}
	drawBox(gc, track, trackStyle)

	knobSize := toggleKnobSize(track)
	if knobSize <= 0 {
		return
	}
	knobTop := track.Min.Y + (track.Dy()-knobSize)/2
	knobLeft := track.Min.X + 4
	if c.checked {
		knobLeft = track.Max.X - knobSize - 4
	}
	if knobLeft < track.Min.X {
		knobLeft = track.Min.X
	}
	knobRect := image.Rect(knobLeft, knobTop, knobLeft+knobSize, knobTop+knobSize)
	knobStyle := Style{Background: c.style.Foreground, BorderColor: c.style.BorderColor, BorderWidth: max(1, c.style.BorderWidth)}
	if knobStyle.Background == "" {
		knobStyle.Background = "#F2F2E9"
	}
	if knobStyle.BorderColor == "" {
		knobStyle.BorderColor = "#D9E2E8FF"
	}
	drawBox(gc, knobRect, knobStyle)

	if c.pressed {
		shade := color.RGBA{255, 255, 255, 36}
		gc.SetFillColor(shade)
		gc.BeginPath()
		gc.MoveTo(float64(track.Min.X), float64(track.Min.Y))
		gc.LineTo(float64(track.Max.X), float64(track.Min.Y))
		gc.LineTo(float64(track.Max.X), float64(track.Max.Y))
		gc.LineTo(float64(track.Min.X), float64(track.Max.Y))
		gc.Close()
		gc.Fill()
	}
}

func toggleTrackRect(bbox image.Rectangle) image.Rectangle {
	content := insetRect(bbox, 4)
	if content.Empty() {
		return image.Rectangle{}
	}

	trackWidth := content.Dx()
	maxWidth := int(float64(content.Dy()) * 2.5)
	if maxWidth > 0 && trackWidth > maxWidth {
		trackWidth = maxWidth
	}
	if trackWidth < 24 {
		trackWidth = content.Dx()
	}
	if trackWidth > content.Dx() {
		trackWidth = content.Dx()
	}

	left := content.Min.X + (content.Dx()-trackWidth)/2
	return image.Rect(left, content.Min.Y, left+trackWidth, content.Max.Y)
}

func toggleKnobSize(track image.Rectangle) int {
	available := minInt(track.Dx()-8, track.Dy()-8)
	if available <= 0 {
		return 0
	}
	if available >= 20 {
		return available
	}
	fallback := minInt(track.Dx()-4, track.Dy()-4)
	if fallback < 0 {
		return 0
	}
	return fallback
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *toggleComponent) BoundingBox() image.Rectangle {
	return c.bbox
}

func (c *toggleComponent) HandleTouch(x, y int, isRelease bool) TouchResult {
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
