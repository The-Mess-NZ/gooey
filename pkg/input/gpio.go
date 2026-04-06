package input

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	periphgpio "periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

type gpioBackend interface {
	OpenButton(spec GPIOButtonSpec) (gpioButton, error)
}

type gpioButton interface {
	State() (bool, error)
	WaitForChange(ctx context.Context) (bool, error)
	Close() error
}

type gpioWatcher struct {
	spec       GPIOButtonSpec
	button     gpioButton
	stableHigh bool
	lastChange time.Time
}

// GPIOListener reads GPIO-backed button state changes and emits normalized events.
type GPIOListener struct {
	handler EventHandler
	buttons []*gpioWatcher
	clock   func() time.Time
}

// NewGPIOListener creates a GPIO listener backed by periph.
func NewGPIOListener(specs []GPIOButtonSpec, handler EventHandler) (*GPIOListener, error) {
	return NewGPIOListenerWithBackend(specs, handler, &periphGPIOBackend{})
}

func NewGPIOListenerWithBackend(specs []GPIOButtonSpec, handler EventHandler, backend gpioBackend) (*GPIOListener, error) {
	if backend == nil {
		return nil, fmt.Errorf("gpio backend is required")
	}

	listener := &GPIOListener{
		handler: handler,
		buttons: make([]*gpioWatcher, 0, len(specs)),
		clock:   time.Now,
	}

	for _, spec := range specs {
		normalized, err := normalizeGPIOButtonSpec(spec)
		if err != nil {
			listener.close()
			return nil, err
		}
		button, err := backend.OpenButton(normalized)
		if err != nil {
			listener.close()
			return nil, err
		}
		listener.buttons = append(listener.buttons, &gpioWatcher{spec: normalized, button: button})
	}

	return listener, nil
}

// Start runs one watcher per configured GPIO button until the context is canceled.
func (l *GPIOListener) Start(ctx context.Context) {
	if l == nil {
		return
	}
	defer l.close()

	if len(l.buttons) == 0 {
		<-ctx.Done()
		return
	}

	log.Printf("Input Engine listening for GPIO button events on %d input(s)", len(l.buttons))
	var wg sync.WaitGroup
	for _, watcher := range l.buttons {
		wg.Add(1)
		go func(w *gpioWatcher) {
			defer wg.Done()
			l.watchButton(ctx, w)
		}(watcher)
	}
	wg.Wait()
	log.Println("GPIO listener shutting down")
}

func (l *GPIOListener) watchButton(ctx context.Context, watcher *gpioWatcher) {
	current, err := watcher.button.State()
	if err != nil {
		log.Printf("GPIO input %s initial read failed: %v", watcher.spec.ID, err)
		return
	}
	watcher.stableHigh = current

	for {
		pressed, err := watcher.button.WaitForChange(ctx)
		log.Printf("GPIO input %s state changed: %t", watcher.spec.ID, pressed)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("GPIO input %s stopped: %v", watcher.spec.ID, err)
			return
		}
		if pressed == watcher.stableHigh {
			continue
		}

		now := l.clock()
		if watcher.spec.Debounce > 0 && !watcher.lastChange.IsZero() && now.Sub(watcher.lastChange) < watcher.spec.Debounce {
			continue
		}

		watcher.stableHigh = pressed
		watcher.lastChange = now

		phase := PhaseRelease
		if pressed {
			phase = PhasePress
		}
		if !shouldEmitGPIOPhase(watcher.spec, phase) || l.handler == nil {
			continue
		}

		l.handler.HandleInputEvent(Event{
			Source:      SourceGPIO,
			InputID:     watcher.spec.ID,
			ControlType: ControlTypeButton,
			Phase:       phase,
		})
	}
}

func (l *GPIOListener) close() {
	if l == nil {
		return
	}
	for _, watcher := range l.buttons {
		if watcher == nil || watcher.button == nil {
			continue
		}
		_ = watcher.button.Close()
	}
}

func normalizeGPIOButtonSpec(spec GPIOButtonSpec) (GPIOButtonSpec, error) {
	spec.ID = strings.TrimSpace(spec.ID)
	spec.Pin = strings.TrimSpace(spec.Pin)
	spec.Pull = strings.ToLower(strings.TrimSpace(spec.Pull))
	if spec.ID == "" {
		return GPIOButtonSpec{}, fmt.Errorf("gpio button id is required")
	}
	if spec.Pin == "" {
		return GPIOButtonSpec{}, fmt.Errorf("gpio button %s pin is required", spec.ID)
	}
	if spec.Debounce < 0 {
		return GPIOButtonSpec{}, fmt.Errorf("gpio button %s debounce must be zero or greater", spec.ID)
	}
	if !spec.EmitPress && !spec.EmitRelease {
		spec.EmitPress = true
		spec.EmitRelease = true
	}
	return spec, nil
}

func shouldEmitGPIOPhase(spec GPIOButtonSpec, phase string) bool {
	switch phase {
	case PhasePress:
		return spec.EmitPress
	case PhaseRelease:
		return spec.EmitRelease
	default:
		return false
	}
}

type periphGPIOBackend struct {
	initOnce sync.Once
	initErr  error
}

func (b *periphGPIOBackend) OpenButton(spec GPIOButtonSpec) (gpioButton, error) {
	if err := b.init(); err != nil {
		return nil, fmt.Errorf("initialize gpio host: %w", err)
	}

	pin := gpioreg.ByName(spec.Pin)
	if pin == nil {
		return nil, fmt.Errorf("gpio button %s pin %q not found", spec.ID, spec.Pin)
	}

	pull, err := parsePeriphPull(spec.Pull)
	if err != nil {
		return nil, fmt.Errorf("gpio button %s: %w", spec.ID, err)
	}
	if err := pin.In(pull, periphgpio.BothEdges); err != nil {
		return nil, fmt.Errorf("gpio button %s pin %q setup failed: %w", spec.ID, spec.Pin, err)
	}

	return &periphGPIOButton{pin: pin, invert: spec.Invert}, nil
}

func (b *periphGPIOBackend) init() error {
	b.initOnce.Do(func() {
		_, b.initErr = host.Init()
	})
	return b.initErr
}

type periphGPIOButton struct {
	pin    periphgpio.PinIO
	invert bool
}

func (b *periphGPIOButton) State() (bool, error) {
	if b == nil || b.pin == nil {
		return false, fmt.Errorf("gpio pin is not initialized")
	}
	return periphLevelPressed(b.pin.Read(), b.invert), nil
}

func (b *periphGPIOButton) WaitForChange(ctx context.Context) (bool, error) {
	if b == nil || b.pin == nil {
		return false, fmt.Errorf("gpio pin is not initialized")
	}
	for {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if b.pin.WaitForEdge(200 * time.Millisecond) {
			return periphLevelPressed(b.pin.Read(), b.invert), nil
		}
	}
}

func (b *periphGPIOButton) Close() error {
	return nil
}

func parsePeriphPull(value string) (periphgpio.Pull, error) {
	switch value {
	case "", "float":
		return periphgpio.Float, nil
	case "up":
		return periphgpio.PullUp, nil
	case "down":
		return periphgpio.PullDown, nil
	default:
		return periphgpio.PullNoChange, fmt.Errorf("unsupported gpio pull %q", value)
	}
}

func periphLevelPressed(level periphgpio.Level, invert bool) bool {
	pressed := level == periphgpio.High
	if invert {
		pressed = !pressed
	}
	return pressed
}
