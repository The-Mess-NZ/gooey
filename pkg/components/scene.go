package components

import (
	"fmt"
	"image"
)

const SceneVersionAlpha1 = "v1alpha1"

const InputControlTypeButton = "button"

// InputBinding maps a stable hardware input ID to scene-specific semantics.
type InputBinding struct {
	ID          string `json:"id"`
	ControlType string `json:"controlType,omitempty"`
	OnPress     string `json:"onPress,omitempty"`
	OnRelease   string `json:"onRelease,omitempty"`
}

const (
	NodeTypeContainer = "container"
	NodeTypeLabel     = "label"
	NodeTypeButton    = "button"
	NodeTypeToggle    = "toggle"

	LayoutDirectionVertical   = "vertical"
	LayoutDirectionHorizontal = "horizontal"
)

// SceneDocument is the host-owned scenegraph submitted to Gooey.
type SceneDocument struct {
	Version       string         `json:"version,omitempty"`
	InputBindings []InputBinding `json:"inputBindings,omitempty"`
	Root          SceneNode      `json:"root"`
}

// SceneNode describes a single renderable or layout node.
type SceneNode struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Bounds   *Rect       `json:"bounds,omitempty"`
	Layout   *Layout     `json:"layout,omitempty"`
	Style    *Style      `json:"style,omitempty"`
	Text     string      `json:"text,omitempty"`
	Action   string      `json:"action,omitempty"`
	Checked  *bool       `json:"checked,omitempty"`
	Visible  *bool       `json:"visible,omitempty"`
	Children []SceneNode `json:"children,omitempty"`
}

// Rect is a JSON-friendly rectangle definition.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Layout controls automatic child placement inside a container node.
type Layout struct {
	Direction string `json:"direction,omitempty"`
	Gap       int    `json:"gap,omitempty"`
	Padding   Insets `json:"padding,omitempty"`
}

// Insets describes top/right/bottom/left padding.
type Insets struct {
	Top    int `json:"top,omitempty"`
	Right  int `json:"right,omitempty"`
	Bottom int `json:"bottom,omitempty"`
	Left   int `json:"left,omitempty"`
}

// Style controls simple visual properties for alpha widgets.
type Style struct {
	Background  string  `json:"background,omitempty"`
	Foreground  string  `json:"foreground,omitempty"`
	BorderColor string  `json:"borderColor,omitempty"`
	BorderWidth int     `json:"borderWidth,omitempty"`
	FontSize    float64 `json:"fontSize,omitempty"`
	TextPadding int     `json:"textPadding,omitempty"`
}

// ComponentPatch applies targeted changes to an existing scene node.
type ComponentPatch struct {
	ID      string  `json:"id"`
	Bounds  *Rect   `json:"bounds,omitempty"`
	Style   *Style  `json:"style,omitempty"`
	Text    *string `json:"text,omitempty"`
	Action  *string `json:"action,omitempty"`
	Checked *bool   `json:"checked,omitempty"`
	Visible *bool   `json:"visible,omitempty"`
}

func (d *SceneDocument) normalize() {
	if d.Version == "" {
		d.Version = SceneVersionAlpha1
	}
	for i := range d.InputBindings {
		d.InputBindings[i].normalize()
	}
}

func (b *InputBinding) normalize() {
	if b.ControlType == "" {
		b.ControlType = InputControlTypeButton
	}
}

func (n SceneNode) isVisible() bool {
	return n.Visible == nil || *n.Visible
}

func (r Rect) toImageRect(origin image.Point) image.Rectangle {
	return image.Rect(origin.X+r.X, origin.Y+r.Y, origin.X+r.X+r.Width, origin.Y+r.Y+r.Height)
}

// InputBindingByID returns the scene-specific binding for a stable hardware input.
func (d SceneDocument) InputBindingByID(id string) (InputBinding, bool) {
	for _, binding := range d.InputBindings {
		if binding.ID == id {
			return binding, true
		}
	}
	return InputBinding{}, false
}

func (i Insets) inset(rect image.Rectangle) image.Rectangle {
	minX := rect.Min.X + i.Left
	minY := rect.Min.Y + i.Top
	maxX := rect.Max.X - i.Right
	maxY := rect.Max.Y - i.Bottom
	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}
	return image.Rect(minX, minY, maxX, maxY)
}

// ApplyPatch mutates a scene node identified by a stable component ID.
func ApplyPatch(doc *SceneDocument, patch ComponentPatch) error {
	if doc == nil {
		return fmt.Errorf("scene is not loaded")
	}
	if patch.ID == "" {
		return fmt.Errorf("patch id is required")
	}

	node, err := findNode(&doc.Root, patch.ID)
	if err != nil {
		return err
	}

	if patch.Bounds != nil {
		node.Bounds = patch.Bounds
	}
	if patch.Style != nil {
		node.Style = patch.Style
	}
	if patch.Text != nil {
		node.Text = *patch.Text
	}
	if patch.Action != nil {
		node.Action = *patch.Action
	}
	if patch.Checked != nil {
		node.Checked = patch.Checked
	}
	if patch.Visible != nil {
		node.Visible = patch.Visible
	}

	return nil
}

func findNode(node *SceneNode, id string) (*SceneNode, error) {
	if node.ID == id {
		return node, nil
	}
	for i := range node.Children {
		found, err := findNode(&node.Children[i], id)
		if err == nil {
			return found, nil
		}
	}
	return nil, fmt.Errorf("component %q not found", id)
}
