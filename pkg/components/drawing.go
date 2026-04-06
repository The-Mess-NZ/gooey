package components

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

func defaultStyle(nodeType string) Style {
	defaults := componentDefaults(nodeType)
	switch nodeType {
	case NodeTypeButton:
		return Style{
			Background:  "#2F6B4F",
			Foreground:  "#F2F2E9",
			BorderColor: "#9ED8B5",
			BorderWidth: defaults.BorderWidth,
			FontSize:    defaults.FontSize,
			TextPadding: defaults.TextPadding,
		}
	case NodeTypeLabel:
		return Style{
			Foreground:  "#F2F2E9",
			FontSize:    defaults.FontSize,
			TextPadding: defaults.TextPadding,
		}
	case NodeTypeToggle:
		return Style{
			Background:  "#2F6B4F",
			Foreground:  "#F2F2E9",
			BorderColor: "#9ED8B5",
			BorderWidth: defaults.BorderWidth,
			TextPadding: defaults.TextPadding,
		}
	case NodeTypeSoftBar:
		return Style{
			Background:  "#314554",
			Foreground:  "#F2F2E9",
			BorderColor: "#A8C2D4",
			BorderWidth: defaults.BorderWidth,
			FontSize:    defaults.FontSize,
			TextPadding: defaults.TextPadding,
		}
	default:
		return Style{
			FontSize:    defaults.FontSize,
			TextPadding: defaults.TextPadding,
			BorderWidth: defaults.BorderWidth,
		}
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
	if override.TextPadding != 0 {
		style.TextPadding = override.TextPadding
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
		fontSize = 13
	}

	padding := style.TextPadding
	content := insetRect(bbox, padding)
	if content.Empty() {
		return
	}

	lineHeight := fontSize * 1.2
	if lineHeight <= 0 {
		lineHeight = 1
	}
	maxLines := int(math.Floor(float64(content.Dy()) / lineHeight))
	if maxLines < 1 {
		maxLines = 1
	}

	// TODO: This 0.65 is a very rough approximation. Could look at using GetStringBounds for better accuracy
	charWidth := math.Max(fontSize*0.65, 1)
	maxChars := int(math.Floor(float64(content.Dx()) / charWidth))
	if maxChars < 1 {
		maxChars = 1
	}

	lines := wrapText(text, maxChars, maxLines)
	if len(lines) == 0 {
		return
	}

	gc.SetFillColor(fg)
	gc.SetFontSize(fontSize)

	totalHeight := float64(len(lines)-1)*lineHeight + fontSize
	y := float64(content.Min.Y) + fontSize
	if centered {
		y = float64(content.Min.Y) + (float64(content.Dy())-totalHeight)/2 + fontSize*0.9
		minBaseline := float64(content.Min.Y) + fontSize
		if y < minBaseline {
			y = minBaseline
		}
	}

	for _, line := range lines {
		x := float64(content.Min.X)
		if centered {
			lineWidth := charWidth * float64(len(line))
			x = float64(content.Min.X) + (float64(content.Dx())-lineWidth)/2
			if x < float64(content.Min.X) {
				x = float64(content.Min.X)
			}
		}

		gc.FillStringAt(line, x, y)
		y += lineHeight
		if y > float64(content.Max.Y)+fontSize*0.25 {
			break
		}
	}
}

func wrapText(text string, maxChars, maxLines int) []string {
	if maxChars <= 0 || maxLines <= 0 {
		return nil
	}

	words := strings.Fields(strings.ReplaceAll(text, "\n", " <NL> "))
	if len(words) == 0 {
		return []string{trimToWidth(strings.TrimSpace(text), maxChars)}
	}

	lines := make([]string, 0, maxLines)
	current := ""
	truncated := false

	appendLine := func(line string) bool {
		if len(lines) == maxLines {
			truncated = true
			return false
		}
		lines = append(lines, trimToWidth(strings.TrimSpace(line), maxChars))
		return true
	}

	for _, word := range words {
		if word == "<NL>" {
			if current == "" {
				if !appendLine("") {
					break
				}
				continue
			}
			if !appendLine(current) {
				break
			}
			current = ""
			continue
		}

		candidate := word
		if current != "" {
			candidate = current + " " + word
		}

		if len(candidate) <= maxChars {
			current = candidate
			continue
		}

		if current != "" {
			if !appendLine(current) {
				break
			}
			current = ""
		}

		for len(word) > maxChars {
			if len(lines) == maxLines-1 {
				truncated = true
				word = trimToWidth(word, maxChars)
				break
			}
			if !appendLine(word[:maxChars]) {
				break
			}
			word = word[maxChars:]
		}
		current = word
	}

	if !truncated && current != "" {
		appendLine(current)
	}

	if truncated && len(lines) > 0 {
		lines[len(lines)-1] = addEllipsis(lines[len(lines)-1], maxChars)
	}

	return lines
}

func trimToWidth(text string, maxChars int) string {
	trimmed := strings.TrimSpace(text)
	if maxChars <= 0 || len(trimmed) <= maxChars {
		return trimmed
	}
	return addEllipsis(trimmed, maxChars)
}

func addEllipsis(text string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	if len(text) <= maxChars {
		return text
	}
	if maxChars == 1 {
		return "."
	}
	if maxChars <= 3 {
		return strings.Repeat(".", maxChars)
	}
	return strings.TrimSpace(text[:maxChars-3]) + "..."
}

func insetRect(rect image.Rectangle, inset int) image.Rectangle {
	if inset <= 0 {
		return rect
	}
	minX := rect.Min.X + inset
	minY := rect.Min.Y + inset
	maxX := rect.Max.X - inset
	maxY := rect.Max.Y - inset
	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}
	return image.Rect(minX, minY, maxX, maxY)
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
