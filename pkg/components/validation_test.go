package components

import "testing"

func TestValidateSceneDocumentAcceptsInputBindings(t *testing.T) {
	doc := SceneDocument{
		InputBindings: []InputBinding{{ID: "next", OnPress: "next_item"}},
		Root:          SceneNode{ID: "root", Type: NodeTypeContainer},
	}

	if err := ValidateSceneDocument(doc); err != nil {
		t.Fatalf("ValidateSceneDocument() error = %v", err)
	}
}

func TestValidateSceneDocumentAcceptsSoftButtonBar(t *testing.T) {
	doc := SceneDocument{
		Root: SceneNode{
			ID:   "root",
			Type: NodeTypeContainer,
			Children: []SceneNode{{
				ID:   "soft-bar",
				Type: NodeTypeSoftBar,
				SoftButtons: []SoftButtonSlot{
					{Label: "<<"},
					{Label: "<"},
					{Label: ">"},
					{Label: ">>"},
				},
			}},
		},
	}

	if err := ValidateSceneDocument(doc); err != nil {
		t.Fatalf("ValidateSceneDocument() error = %v", err)
	}
}

func TestValidateSceneDocumentRejectsInvalidInputBindings(t *testing.T) {
	tests := []struct {
		name string
		doc  SceneDocument
	}{
		{
			name: "missing id",
			doc: SceneDocument{
				InputBindings: []InputBinding{{OnPress: "next_item"}},
				Root:          SceneNode{ID: "root", Type: NodeTypeContainer},
			},
		},
		{
			name: "duplicate id",
			doc: SceneDocument{
				InputBindings: []InputBinding{{ID: "next", OnPress: "next_item"}, {ID: "next", OnRelease: "back"}},
				Root:          SceneNode{ID: "root", Type: NodeTypeContainer},
			},
		},
		{
			name: "unsupported control type",
			doc: SceneDocument{
				InputBindings: []InputBinding{{ID: "dial", ControlType: "encoder", OnPress: "next_item"}},
				Root:          SceneNode{ID: "root", Type: NodeTypeContainer},
			},
		},
		{
			name: "missing actions",
			doc: SceneDocument{
				InputBindings: []InputBinding{{ID: "next"}},
				Root:          SceneNode{ID: "root", Type: NodeTypeContainer},
			},
		},
		{
			name: "soft bar requires four slots",
			doc: SceneDocument{
				Root: SceneNode{
					ID:   "root",
					Type: NodeTypeContainer,
					Children: []SceneNode{{
						ID:          "soft-bar",
						Type:        NodeTypeSoftBar,
						SoftButtons: []SoftButtonSlot{{Label: "prev"}, {Label: "next"}},
					}},
				},
			},
		},
		{
			name: "soft bar fields rejected on other nodes",
			doc: SceneDocument{
				Root: SceneNode{
					ID:          "root",
					Type:        NodeTypeLabel,
					SoftButtons: []SoftButtonSlot{{Label: "prev"}, {Label: "next"}, {Label: "ports"}, {Label: ""}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateSceneDocument(test.doc); err == nil {
				t.Fatalf("ValidateSceneDocument() error = nil, want failure")
			}
		})
	}
}
