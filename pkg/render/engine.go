package render

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
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
	components []components.Component

	redrawChan chan struct{}
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

// HandleTouch maps the input X/Y to visual feedback. It safely iterates
// backwards (top-most component first) over the scenegraph checking bounds.
func (e *Engine) HandleTouch(x, y int, isRelease bool) {
	e.mu.Lock()
	needsRedraw := false
	// Process top-down
	for i := len(e.components) - 1; i >= 0; i-- {
		comp := e.components[i]
		if comp.HandleTouch(x, y, isRelease) {
			needsRedraw = true
		}
	}
	e.mu.Unlock()

	if needsRedraw {
		e.TriggerRedraw()
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
