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

// TouchListener connects to the raw Linux evdev input device,
// manages calibration ranges, and converts raw ADC bounds to screen pixels.
type TouchListener struct {
	devicePath string
	dev        *evdev.InputDevice
	handler    TouchHandler

	// Calibration data - replace with actual values based on hardware ADC limits
	MinX, MaxX int32
	MinY, MaxY int32

	ScreenWidth  int
	ScreenHeight int

	InvertX bool
	InvertY bool
	SwapXY  bool
}

// NewTouchListener binds to the specified evdev path and sets default calibration limits
func NewTouchListener(devicePath string, handler TouchHandler) (*TouchListener, error) {
	dev, err := evdev.Open(devicePath)
	if err != nil {
		return nil, err
	}

	return &TouchListener{
		devicePath:   devicePath,
		dev:          dev,
		handler:      handler,
		MinX:         370,  // Typical raw base
		MaxX:         2300, // Typical raw ceiling
		MinY:         345,
		MaxY:         3740,
		ScreenWidth:  320,
		ScreenHeight: 240,
		InvertX:      false,
		InvertY:      false,
		SwapXY:       true,
	}, nil
}

// Start runs an infinite blocking loop consuming evdev structs and dispatching to TouchHandler
func (t *TouchListener) Start(ctx context.Context) {
	log.Printf("Input Engine listening for touch events on %s", t.devicePath)

	// Because evdev ReadOne is blocking but we want context cancellation,
	// we will rely on device closure to break the read loop out if needed.
	go func() {
		<-ctx.Done()
		t.dev.Close()
		log.Println("Touch listener shutting down")
	}()

	var rawX, rawY int32
	touchDown := false
	hasTouchState := false
	pendingTouchState := false
	nextTouchDown := false

	for {
		ev, err := t.dev.ReadOne()
		if err != nil {
			// Expected on close/shutdown
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
			if pendingTouchState && t.handler != nil {
				touchDown = nextTouchDown
				pendingTouchState = false
				x, y := t.transformCoordinates(rawX, rawY)
				log.Printf("Touch event at (%d, %d), release=%t", x, y, !touchDown)
				t.handler.HandleTouch(x, y, !touchDown)
			}
		}
	}
}

// transformCoordinates converts the raw 12-bit ADC to absolute screen pixels
func (t *TouchListener) transformCoordinates(rawX, rawY int32) (int, int) {
	x := t.scale(rawX, t.MinX, t.MaxX, t.ScreenWidth)
	y := t.scale(rawY, t.MinY, t.MaxY, t.ScreenHeight)

	if t.SwapXY {
		x, y = y, x
	}
	if t.InvertX {
		x = t.ScreenWidth - x
	}
	if t.InvertY {
		y = t.ScreenHeight - y
	}

	return x, y
}

func (t *TouchListener) scale(val int32, min, max int32, screenDim int) int {
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
	scaled := float32(val-min) / float32(rangeADC) * float32(screenDim)
	return int(scaled)
}
