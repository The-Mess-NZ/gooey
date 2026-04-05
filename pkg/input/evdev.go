package input

import (
	"context"
	"log"

	evdev "github.com/holoplot/go-evdev"
)

// TouchHandler defines an interface for processing interpreted touch events
type TouchHandler interface {
	HandleTouch(x, y int, isRelease bool)
}

// RawTouchHandler consumes raw touch samples before calibration is applied.
type RawTouchHandler interface {
	HandleRawTouch(rawX, rawY int32, isRelease bool)
}

// TouchListener connects to the raw Linux evdev input device,
// manages calibration ranges, and converts raw ADC bounds to screen pixels.
// TODO: Initialize with a config object?
type TouchListener struct {
	devicePath string
	dev        *evdev.InputDevice
	handler    TouchHandler

	// Raw X ADC bound. This in the perspective of the touch controller (e.g. evtest output).
	MinXRaw, MaxXRaw int32
	// Raw Y ADC bound. This in the perspective of the touch controller (e.g. evtest output).
	MinYRaw, MaxYRaw int32

	// Physical screen X dimensions in pixels, in the touch controller's perspective.
	ScreenXPixels int
	// Physical screen Y dimensions in pixels, in the touch controller's perspective.
	ScreenYPixels int

	// Invert raw X values if the touch controller is mounted in reverse.
	// For example, if the touch controller is rotated 180 degrees, and "left" is a higher ADC value than "right", then InvertX should be true.
	InvertX bool
	// Invert raw Y values if the touch controller is mounted in reverse.
	// For example, if the touch controller is rotated 180 degrees, and "down" is a higher ADC value than "up", then InvertX should be true.
	InvertY bool

	// IsLandscape indicates whether the display is mounted in landscape orientation, which swaps the X/Y axes.
	IsLandscape bool
}

// RawTouchListener reads raw touch values from evdev without coordinate transforms.
type RawTouchListener struct {
	devicePath string
	dev        *evdev.InputDevice
	handler    RawTouchHandler
}

// NewTouchListener binds to the specified evdev path and sets calibration parameters to 0.
func NewTouchListener(devicePath string, handler TouchHandler) (*TouchListener, error) {
	dev, err := evdev.Open(devicePath)
	if err != nil {
		return nil, err
	}

	return &TouchListener{
		devicePath:    devicePath,
		dev:           dev,
		handler:       handler,
		MinXRaw:       0,
		MaxXRaw:       0,
		MinYRaw:       0,
		MaxYRaw:       0,
		ScreenXPixels: 0,
		ScreenYPixels: 0,
		InvertX:       false,
		InvertY:       false,
		IsLandscape:   false,
	}, nil
}

// NewRawTouchListener binds to the specified evdev path and emits raw touch samples.
func NewRawTouchListener(devicePath string, handler RawTouchHandler) (*RawTouchListener, error) {
	dev, err := evdev.Open(devicePath)
	if err != nil {
		return nil, err
	}

	return &RawTouchListener{
		devicePath: devicePath,
		dev:        dev,
		handler:    handler,
	}, nil
}

// Start runs an infinite blocking loop consuming evdev structs and dispatching to TouchHandler
func (t *TouchListener) Start(ctx context.Context) {
	log.Printf("Input Engine listening for touch events on %s", t.devicePath)
	runTouchLoop(ctx, t.dev, func(rawX, rawY int32, isRelease bool) {
		if t.handler == nil {
			return
		}
		x, y := t.transformCoordinates(rawX, rawY)
		log.Printf("Touch event at (%d, %d), release=%t", x, y, isRelease)
		t.handler.HandleTouch(x, y, isRelease)
	})
}

// Start runs the raw touch loop and emits uncalibrated raw coordinates.
func (t *RawTouchListener) Start(ctx context.Context) {
	log.Printf("Raw touch listener active on %s", t.devicePath)
	runTouchLoop(ctx, t.dev, func(rawX, rawY int32, isRelease bool) {
		if t.handler != nil {
			t.handler.HandleRawTouch(rawX, rawY, isRelease)
		}
	})
}

// transformCoordinates converts the raw 12-bit ADC to absolute screen pixels
func (t *TouchListener) transformCoordinates(rawX, rawY int32) (int, int) {
	scaledX := t.scale(rawX, t.MinXRaw, t.MaxXRaw, t.ScreenXPixels)
	scaledY := t.scale(rawY, t.MinYRaw, t.MaxYRaw, t.ScreenYPixels)

	if t.InvertX {
		scaledX = t.ScreenXPixels - scaledX
	}
	if t.InvertY {
		scaledY = t.ScreenYPixels - scaledY
	}
	if t.IsLandscape {
		scaledX, scaledY = scaledY, scaledX
	}

	return scaledX, scaledY
}

// scale maps a raw ADC val to screen coordinates based on raw min/max calibration bounds.
func (t *TouchListener) scale(val int32, min, max int32, screenDimension int) int {
	if val < min {
		val = min
	}
	if val > max {
		val = max
	}
	rangeADC := max - min
	if rangeADC == 0 {
		return 0
	}
	scaled := float32(val-min) / float32(rangeADC) * float32(screenDimension)
	return int(scaled)
}

// runTouchLoop continuously reads raw evdev events and dispatches touch samples to the provided handler.
func runTouchLoop(ctx context.Context, dev *evdev.InputDevice, handle func(rawX, rawY int32, isRelease bool)) {
	go func() {
		<-ctx.Done()
		dev.Close()
		log.Println("Touch listener shutting down")
	}()

	var rawX, rawY int32
	touchDown := false
	hasTouchState := false
	pendingTouchState := false
	nextTouchDown := false

	for {
		ev, err := dev.ReadOne()
		if err != nil {
			break
		}

		if ev.Type == evdev.EV_ABS {
			switch ev.Code {
			case evdev.ABS_X:
				rawX = ev.Value
			case evdev.ABS_Y:
				rawY = ev.Value
			case evdev.ABS_MT_POSITION_X:
				rawX = ev.Value
			case evdev.ABS_MT_POSITION_Y:
				rawY = ev.Value
			case evdev.ABS_PRESSURE:
				down := ev.Value > 0
				if !hasTouchState || down != touchDown {
					hasTouchState = true
					pendingTouchState = true
					nextTouchDown = down
				}
			}
		} else if ev.Type == evdev.EV_KEY && ev.Code == evdev.BTN_TOUCH {
			down := ev.Value != 0
			if !hasTouchState || down != touchDown {
				hasTouchState = true
				pendingTouchState = true
				nextTouchDown = down
			}
		} else if ev.Type == evdev.EV_SYN && ev.Code == evdev.SYN_REPORT {
			if pendingTouchState {
				touchDown = nextTouchDown
				pendingTouchState = false
				if handle != nil {
					handle(rawX, rawY, !touchDown)
				}
			}
		}
	}
}
