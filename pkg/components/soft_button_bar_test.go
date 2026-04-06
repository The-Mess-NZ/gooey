package components

import (
	"image"
	"testing"
)

func TestBuildSceneCreatesSoftButtonBar(t *testing.T) {
	doc := SceneDocument{
		Root: SceneNode{
			ID:   "root",
			Type: NodeTypeContainer,
			Layout: &Layout{
				Direction: LayoutDirectionVertical,
			},
			Children: []SceneNode{{
				ID:          "soft-bar",
				Type:        NodeTypeSoftBar,
				Bounds:      &Rect{Height: 32},
				SoftButtons: []SoftButtonSlot{{Label: "<<"}, {Label: "<"}, {Label: ">"}, {Label: ">>"}},
			}},
		},
	}

	components, err := BuildScene(doc, image.Rect(0, 0, 320, 240))
	if err != nil {
		t.Fatalf("BuildScene() error = %v", err)
	}

	bar := findComponentByID(components, "soft-bar")
	if bar == nil {
		t.Fatal("BuildScene() did not return the soft button bar component")
	}
	if got := bar.BoundingBox().Dx(); got != 320 {
		t.Fatalf("soft button bar width = %d, want 320", got)
	}
	if got := bar.BoundingBox().Dy(); got != 32 {
		t.Fatalf("soft button bar height = %d, want 32", got)
	}
}

func TestBuildSceneKeepsSoftButtonBarHeightWithBlankLabels(t *testing.T) {
	withLabels := SceneDocument{
		Root: SceneNode{
			ID:   "root-labeled",
			Type: NodeTypeContainer,
			Layout: &Layout{
				Direction: LayoutDirectionVertical,
			},
			Children: []SceneNode{{
				ID:          "soft-bar-labeled",
				Type:        NodeTypeSoftBar,
				Bounds:      &Rect{Height: 34},
				SoftButtons: []SoftButtonSlot{{Label: "ports"}, {Label: "prev"}, {Label: "next"}, {Label: ""}},
			}},
		},
	}
	blankLabels := SceneDocument{
		Root: SceneNode{
			ID:   "root-blank",
			Type: NodeTypeContainer,
			Layout: &Layout{
				Direction: LayoutDirectionVertical,
			},
			Children: []SceneNode{{
				ID:          "soft-bar-blank",
				Type:        NodeTypeSoftBar,
				Bounds:      &Rect{Height: 34},
				SoftButtons: []SoftButtonSlot{{Label: ""}, {Label: ""}, {Label: ""}, {Label: ""}},
			}},
		},
	}

	labeledComponents, err := BuildScene(withLabels, image.Rect(0, 0, 320, 240))
	if err != nil {
		t.Fatalf("BuildScene(withLabels) error = %v", err)
	}
	blankComponents, err := BuildScene(blankLabels, image.Rect(0, 0, 320, 240))
	if err != nil {
		t.Fatalf("BuildScene(blankLabels) error = %v", err)
	}

	labeledBar := findComponentByID(labeledComponents, "soft-bar-labeled")
	blankBar := findComponentByID(blankComponents, "soft-bar-blank")
	if labeledBar == nil || blankBar == nil {
		t.Fatal("BuildScene() did not return both soft button bar components")
	}
	if labeledBar.BoundingBox().Dy() != blankBar.BoundingBox().Dy() {
		t.Fatalf("soft button bar heights differ: labeled=%d blank=%d", labeledBar.BoundingBox().Dy(), blankBar.BoundingBox().Dy())
	}
}

func TestSoftButtonBarIgnoresTouch(t *testing.T) {
	bar := &softButtonBarComponent{bbox: image.Rect(0, 0, 320, 32)}
	if result := bar.HandleTouch(10, 10, false); result.NeedsRedraw || len(result.Interactions) > 0 {
		t.Fatalf("HandleTouch(press) = %+v, want no effect", result)
	}
	if result := bar.HandleTouch(10, 10, true); result.NeedsRedraw || len(result.Interactions) > 0 {
		t.Fatalf("HandleTouch(release) = %+v, want no effect", result)
	}
}

func findComponentByID(components []Component, id string) Component {
	for _, component := range components {
		if component.ID() == id {
			return component
		}
	}
	return nil
}
