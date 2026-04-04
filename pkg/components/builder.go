package components

import (
	"fmt"
	"image"
)

// BuildScene compiles a host-owned scene document into renderable components.
func BuildScene(doc SceneDocument, viewport image.Rectangle) ([]Component, error) {
	doc.normalize()
	if doc.Root.ID == "" {
		return nil, fmt.Errorf("scene root id is required")
	}
	if doc.Root.Type == "" {
		return nil, fmt.Errorf("scene root type is required")
	}

	ids := map[string]struct{}{}
	components := make([]Component, 0)
	rootRect := viewport
	if doc.Root.Bounds != nil && doc.Root.Bounds.Width > 0 && doc.Root.Bounds.Height > 0 {
		rootRect = doc.Root.Bounds.toImageRect(viewport.Min)
	}

	if err := buildNode(doc.Root, rootRect, &components, ids); err != nil {
		return nil, err
	}

	return components, nil
}

func buildNode(node SceneNode, rect image.Rectangle, out *[]Component, ids map[string]struct{}) error {
	if !node.isVisible() {
		return nil
	}
	if _, exists := ids[node.ID]; exists {
		return fmt.Errorf("duplicate component id %q", node.ID)
	}
	ids[node.ID] = struct{}{}

	style := mergeStyle(node.Type, node.Style)

	switch node.Type {
	case NodeTypeContainer:
		*out = append(*out, newContainerComponent(node.ID, rect, style))
		childRects := layoutChildren(node, rect)
		for i := range node.Children {
			if err := buildNode(node.Children[i], childRects[i], out, ids); err != nil {
				return err
			}
		}
	case NodeTypeLabel:
		*out = append(*out, newLabelComponent(node.ID, rect, node.Text, style))
	case NodeTypeButton:
		*out = append(*out, newButtonComponent(node.ID, rect, node.Text, node.Action, style))
	default:
		return fmt.Errorf("unsupported component type %q", node.Type)
	}

	return nil
}

func layoutChildren(node SceneNode, parent image.Rectangle) []image.Rectangle {
	if len(node.Children) == 0 {
		return nil
	}

	layout := Layout{}
	if node.Layout != nil {
		layout = *node.Layout
	}
	content := layout.Padding.inset(parent)
	rects := make([]image.Rectangle, len(node.Children))

	if layout.Direction != LayoutDirectionVertical && layout.Direction != LayoutDirectionHorizontal {
		for i, child := range node.Children {
			if child.Bounds != nil && child.Bounds.Width > 0 && child.Bounds.Height > 0 {
				rects[i] = child.Bounds.toImageRect(content.Min)
				continue
			}
			rects[i] = content
		}
		return rects
	}

	axisAvailable := content.Dy()
	crossAvailable := content.Dx()
	if layout.Direction == LayoutDirectionHorizontal {
		axisAvailable = content.Dx()
		crossAvailable = content.Dy()
	}

	gapTotal := max(0, len(node.Children)-1) * layout.Gap
	axisAvailable -= gapTotal
	if axisAvailable < 0 {
		axisAvailable = 0
	}

	autoCount := 0
	reserved := 0
	for _, child := range node.Children {
		size := requestedAxisSize(child, layout.Direction)
		if size > 0 {
			reserved += size
			continue
		}
		autoCount++
	}

	remaining := axisAvailable - reserved
	if remaining < 0 {
		remaining = 0
	}
	autoSize := 0
	extra := 0
	if autoCount > 0 {
		autoSize = remaining / autoCount
		extra = remaining % autoCount
	}

	cursor := 0
	for i, child := range node.Children {
		axisSize := requestedAxisSize(child, layout.Direction)
		if axisSize <= 0 {
			axisSize = autoSize
			if extra > 0 {
				axisSize++
				extra--
			}
		}
		crossSize := requestedCrossSize(child, layout.Direction)
		if crossSize <= 0 || crossSize > crossAvailable {
			crossSize = crossAvailable
		}

		if layout.Direction == LayoutDirectionVertical {
			rects[i] = image.Rect(content.Min.X, content.Min.Y+cursor, content.Min.X+crossSize, content.Min.Y+cursor+axisSize)
		} else {
			rects[i] = image.Rect(content.Min.X+cursor, content.Min.Y, content.Min.X+cursor+axisSize, content.Min.Y+crossSize)
		}
		cursor += axisSize + layout.Gap
	}

	return rects
}

func requestedAxisSize(node SceneNode, direction string) int {
	if node.Bounds == nil {
		return 0
	}
	if direction == LayoutDirectionHorizontal {
		return node.Bounds.Width
	}
	return node.Bounds.Height
}

func requestedCrossSize(node SceneNode, direction string) int {
	if node.Bounds == nil {
		return 0
	}
	if direction == LayoutDirectionHorizontal {
		return node.Bounds.Height
	}
	return node.Bounds.Width
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
