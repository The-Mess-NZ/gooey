package components

import "fmt"

const (
	ValidationCodeInvalidScene = "invalid_scene"
	ValidationCodeInvalidPatch = "invalid_patch"
)

// ValidationError describes a rejected scene or patch payload.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// ValidateSceneDocument checks that a scene document matches the current alpha contract.
func ValidateSceneDocument(doc SceneDocument) error {
	if doc.Version != "" && doc.Version != SceneVersionAlpha1 {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("unsupported scene version %q", doc.Version)}
	}

	ids := map[string]struct{}{}
	if err := validateNode(doc.Root, ids, true); err != nil {
		return err
	}
	return nil
}

// ValidateComponentPatch checks that a patch payload is structurally valid.
func ValidateComponentPatch(patch ComponentPatch) error {
	if patch.ID == "" {
		return &ValidationError{Code: ValidationCodeInvalidPatch, Message: "patch id is required"}
	}
	if patch.Bounds == nil && patch.Style == nil && patch.Text == nil && patch.Action == nil && patch.Visible == nil {
		return &ValidationError{Code: ValidationCodeInvalidPatch, Message: "patch must include at least one field to update"}
	}
	if patch.Bounds != nil {
		if patch.Bounds.Width < 0 {
			return &ValidationError{Code: ValidationCodeInvalidPatch, Message: "patch bounds width must be zero or greater"}
		}
		if patch.Bounds.Height < 0 {
			return &ValidationError{Code: ValidationCodeInvalidPatch, Message: "patch bounds height must be zero or greater"}
		}
	}
	if patch.Style != nil {
		if err := validateStyle(*patch.Style, ValidationCodeInvalidPatch); err != nil {
			return err
		}
	}
	return nil
}

func validateNode(node SceneNode, ids map[string]struct{}, isRoot bool) error {
	if node.ID == "" {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: "scene node id is required"}
	}
	if _, exists := ids[node.ID]; exists {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("duplicate component id %q", node.ID)}
	}
	ids[node.ID] = struct{}{}

	if node.Type == "" {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q type is required", node.ID)}
	}
	if _, ok := componentFactory(node.Type); !ok {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q uses unsupported component type %q", node.ID, node.Type)}
	}
	if isRoot && node.Visible != nil && !*node.Visible {
		return &ValidationError{Code: ValidationCodeInvalidScene, Message: "scene root cannot be hidden"}
	}
	if node.Bounds != nil {
		if node.Bounds.Width < 0 {
			return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q width must be zero or greater", node.ID)}
		}
		if node.Bounds.Height < 0 {
			return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q height must be zero or greater", node.ID)}
		}
	}
	if node.Layout != nil {
		if node.Layout.Direction != "" && node.Layout.Direction != LayoutDirectionVertical && node.Layout.Direction != LayoutDirectionHorizontal {
			return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q layout direction %q is not supported", node.ID, node.Layout.Direction)}
		}
		if node.Layout.Gap < 0 {
			return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q layout gap must be zero or greater", node.ID)}
		}
		if node.Layout.Padding.Top < 0 || node.Layout.Padding.Right < 0 || node.Layout.Padding.Bottom < 0 || node.Layout.Padding.Left < 0 {
			return &ValidationError{Code: ValidationCodeInvalidScene, Message: fmt.Sprintf("scene node %q layout padding must be zero or greater", node.ID)}
		}
	}
	if err := validateStyleValue(node.ID, node.Style, ValidationCodeInvalidScene); err != nil {
		return err
	}

	for _, child := range node.Children {
		if err := validateNode(child, ids, false); err != nil {
			return err
		}
	}
	return nil
}

func validateStyleValue(nodeID string, style *Style, code string) error {
	if style == nil {
		return nil
	}
	if err := validateStyle(*style, code); err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			validationErr.Message = fmt.Sprintf("scene node %q %s", nodeID, validationErr.Message)
		}
		return err
	}
	return nil
}

func validateStyle(style Style, code string) error {
	if style.Background != "" {
		if _, ok := parseHexColor(style.Background); !ok {
			return &ValidationError{Code: code, Message: fmt.Sprintf("style background %q is not a valid hex color", style.Background)}
		}
	}
	if style.Foreground != "" {
		if _, ok := parseHexColor(style.Foreground); !ok {
			return &ValidationError{Code: code, Message: fmt.Sprintf("style foreground %q is not a valid hex color", style.Foreground)}
		}
	}
	if style.BorderColor != "" {
		if _, ok := parseHexColor(style.BorderColor); !ok {
			return &ValidationError{Code: code, Message: fmt.Sprintf("style borderColor %q is not a valid hex color", style.BorderColor)}
		}
	}
	if style.BorderWidth < 0 {
		return &ValidationError{Code: code, Message: "style borderWidth must be zero or greater"}
	}
	if style.FontSize < 0 {
		return &ValidationError{Code: code, Message: "style fontSize must be zero or greater"}
	}
	if style.TextPadding < 0 {
		return &ValidationError{Code: code, Message: "style textPadding must be zero or greater"}
	}
	return nil
}
