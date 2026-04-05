package render

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"

	"github.com/The-Mess-NZ/gooey/pkg/components"
	"github.com/gonutz/framebuffer"
	"github.com/llgcode/draw2d/draw2dimg"
)

// Engine abstracts the framebuffer device and handles the event-driven render loop.
// It manages the main RGBA image context and copies it out to the SPI device interface.
type Engine struct {
	mu         sync.Mutex // Protects the component tree and shared draw context
	fb         *framebuffer.Device
	bounds     image.Rectangle
	canvas     *image.RGBA
	gc         *draw2dimg.GraphicContext
	scene      *components.SceneDocument
	components []components.Component

	redrawChan         chan struct{}
	interactionHandler func(components.Interaction)
}

// StatusSnapshot describes the current runtime scene state.
type StatusSnapshot struct {
	SceneLoaded    bool
	SceneVersion   string
	RootID         string
	ComponentCount int
}

// NewEngine opens the framebuffer and prepares the draw context.
func NewEngine(devicePath string) (*Engine, error) {
	fb, err := framebuffer.Open(devicePath)
	if err != nil {
		return nil, err
	}

	fbnd := fb.Bounds()
	canvas := image.NewRGBA(fbnd)
	gc := draw2dimg.NewGraphicContext(canvas)

	log.Printf("Render Engine initialized on %s (WxH: %v)", devicePath, fbnd.Size())

	return &Engine{
		fb:         fb,
		bounds:     fbnd,
		canvas:     canvas,
		gc:         gc,
		components: []components.Component{},
		redrawChan: make(chan struct{}, 1), // Non-blocking trigger ring
	}, nil
}

// Close gracefully releases the framebuffer file descriptor.
func (e *Engine) Close() {
	e.fb.Close()
}

// SetComponents loads a new top-level scenegraph representation.
func (e *Engine) SetComponents(comps []components.Component) {
	e.mu.Lock()
	e.components = comps
	e.mu.Unlock()
	e.TriggerRedraw()
}

// LoadScene installs a host-owned scene document and rebuilds renderable components.
func (e *Engine) LoadScene(doc components.SceneDocument) error {
	comps, err := components.BuildScene(doc, e.bounds)
	if err != nil {
		return err
	}

	docCopy := doc
	e.mu.Lock()
	e.scene = &docCopy
	e.components = comps
	e.mu.Unlock()
	e.TriggerRedraw()
	return nil
}

// PatchComponent mutates a single scene node and rebuilds the render list.
func (e *Engine) PatchComponent(patch components.ComponentPatch) error {
	e.mu.Lock()
	if e.scene == nil {
		e.mu.Unlock()
		return fmt.Errorf("scene is not loaded")
	}
	if err := components.ApplyPatch(e.scene, patch); err != nil {
		e.mu.Unlock()
		return err
	}
	comps, err := components.BuildScene(*e.scene, e.bounds)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.components = comps
	e.mu.Unlock()
	e.TriggerRedraw()
	return nil
}

// SetInteractionHandler installs a callback for host-facing touch/component events.
func (e *Engine) SetInteractionHandler(handler func(components.Interaction)) {
	e.mu.Lock()
	e.interactionHandler = handler
	e.mu.Unlock()
}

// Snapshot returns the current runtime scene status.
func (e *Engine) Snapshot() StatusSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	snapshot := StatusSnapshot{ComponentCount: len(e.components)}
	if e.scene == nil {
		return snapshot
	}
	snapshot.SceneLoaded = true
	snapshot.SceneVersion = e.scene.Version
	snapshot.RootID = e.scene.Root.ID
	return snapshot
}

// HandleTouch maps the input X/Y to visual feedback. It safely iterates
// backwards (top-most component first) over the scenegraph checking bounds.
func (e *Engine) HandleTouch(x, y int, isRelease bool) {
	e.mu.Lock()
	needsRedraw := false
	interactions := make([]components.Interaction, 0, 2)
	var topHit image.Rectangle
	topHitID := ""
	// Process top-down
	for i := len(e.components) - 1; i >= 0; i-- {
		comp := e.components[i]
		if topHitID == "" && image.Pt(x, y).In(comp.BoundingBox()) {
			topHit = comp.BoundingBox()
			topHitID = comp.ID()
		}
		result := comp.HandleTouch(x, y, isRelease)
		if result.NeedsRedraw {
			needsRedraw = true
		}
		if len(result.Interactions) > 0 {
			interactions = append(interactions, result.Interactions...)
		}
	}
	handler := e.interactionHandler
	e.mu.Unlock()

	if needsRedraw {
		e.TriggerRedraw()
	}

	if handler == nil {
		return
	}

	kind := "touch_press"
	if isRelease {
		kind = "touch_release"
	}
	raw := components.Interaction{Kind: kind, ComponentID: topHitID, X: x, Y: y}
	if topHitID != "" {
		raw.LocalX = x - topHit.Min.X
		raw.LocalY = y - topHit.Min.Y
	}
	handler(raw)
	for _, interaction := range interactions {
		handler(interaction)
	}
}

// TriggerRedraw queues an asynchronous redraw. It drops duplicate contiguous signals.
func (e *Engine) TriggerRedraw() {
	select {
	case e.redrawChan <- struct{}{}:
	default:
		// A redraw is already queued; continue.
	}
}

// Loop runs an event-driven render loop blocking on Context done or redraw queues.
// Because Midipunk handles midi traffic, we only redraw when state actually changes.
func (e *Engine) Loop(ctx context.Context) {
	// Let's do an initial draw immediately so the user doesn't stare at a blank screen.
	e.TriggerRedraw()

	for {
		select {
		case <-ctx.Done():
			log.Println("Render Engine shut down requested. Halting loop.")
			return
		case <-e.redrawChan:
			e.drawFrame()
		}
	}
}

// drawFrame locks the scene, clears the canvas, applies all Component Draw calls,
// and pushes the canvas buffer to the physical `/dev/fb0` memory.
func (e *Engine) drawFrame() {
	e.mu.Lock()
	// Clear the canvas to black (or background color) before rendering depth
	draw.Draw(e.canvas, e.bounds, &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)

	// Render Scenegraph
	for _, comp := range e.components {
		comp.Draw(e.gc)
	}

	// Dump final frame to the hardware descriptor
	draw.Draw(e.fb, e.bounds, e.canvas, image.Pt(0, 0), draw.Src)
	e.mu.Unlock()
}
