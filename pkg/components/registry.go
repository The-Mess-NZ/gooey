package components

import (
	"fmt"
	"image"
	"sync"
)

// Factory constructs a renderable component for a scene node.
type Factory func(node SceneNode, rect image.Rectangle, style Style) (Component, error)

var registeredComponents = struct {
	mu        sync.RWMutex
	factories map[string]Factory
}{
	factories: map[string]Factory{},
}

// RegisterComponent registers a component factory for a scene node type.
func RegisterComponent(nodeType string, factory Factory) {
	if nodeType == "" {
		panic("component node type is required")
	}
	if factory == nil {
		panic("component factory is required")
	}

	registeredComponents.mu.Lock()
	defer registeredComponents.mu.Unlock()
	if _, exists := registeredComponents.factories[nodeType]; exists {
		panic(fmt.Sprintf("component %q already registered", nodeType))
	}
	registeredComponents.factories[nodeType] = factory
}

func componentFactory(nodeType string) (Factory, bool) {
	registeredComponents.mu.RLock()
	defer registeredComponents.mu.RUnlock()
	factory, ok := registeredComponents.factories[nodeType]
	return factory, ok
}
