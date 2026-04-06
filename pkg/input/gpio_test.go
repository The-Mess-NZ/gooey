package input

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeGPIOBackend struct {
	buttons map[string]*fakeGPIOButton
}

func (b *fakeGPIOBackend) OpenButton(spec GPIOButtonSpec) (gpioButton, error) {
	button, ok := b.buttons[spec.ID]
	if !ok {
		return nil, fmt.Errorf("unknown button %s", spec.ID)
	}
	return button, nil
}

type fakeGPIOButton struct {
	initial bool
	changes chan bool
}

func (b *fakeGPIOButton) State() (bool, error) {
	return b.initial, nil
}

func (b *fakeGPIOButton) WaitForChange(ctx context.Context) (bool, error) {
	select {
	case value := <-b.changes:
		return value, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (b *fakeGPIOButton) Close() error {
	return nil
}

type collectingHandler struct {
	mu     sync.Mutex
	events []Event
	ch     chan Event
}

func newCollectingHandler(buffer int) *collectingHandler {
	return &collectingHandler{ch: make(chan Event, buffer)}
}

func (h *collectingHandler) HandleInputEvent(event Event) {
	h.mu.Lock()
	h.events = append(h.events, event)
	h.mu.Unlock()
	h.ch <- event
}

func (h *collectingHandler) snapshot() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	cloned := make([]Event, len(h.events))
	copy(cloned, h.events)
	return cloned
}

func TestGPIOListenerEmitsConfiguredPhases(t *testing.T) {
	handler := newCollectingHandler(2)
	button := &fakeGPIOButton{changes: make(chan bool, 2)}
	listener, err := NewGPIOListenerWithBackend([]GPIOButtonSpec{{
		ID:          "next",
		Pin:         "17",
		EmitPress:   true,
		EmitRelease: false,
	}}, handler, &fakeGPIOBackend{buttons: map[string]*fakeGPIOButton{"next": button}})
	if err != nil {
		t.Fatalf("NewGPIOListenerWithBackend() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		listener.Start(ctx)
		close(done)
	}()

	button.changes <- true
	select {
	case event := <-handler.ch:
		if event.Phase != PhasePress {
			t.Fatalf("phase = %q, want %q", event.Phase, PhasePress)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("timed out waiting for press event")
	}

	button.changes <- false
	select {
	case event := <-handler.ch:
		cancel()
		t.Fatalf("unexpected extra event: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	<-done
}

func TestGPIOListenerDebouncesRapidToggle(t *testing.T) {
	handler := newCollectingHandler(4)
	button := &fakeGPIOButton{changes: make(chan bool, 4)}
	listener, err := NewGPIOListenerWithBackend([]GPIOButtonSpec{{
		ID:          "next",
		Pin:         "17",
		Debounce:    50 * time.Millisecond,
		EmitPress:   true,
		EmitRelease: true,
	}}, handler, &fakeGPIOBackend{buttons: map[string]*fakeGPIOButton{"next": button}})
	if err != nil {
		t.Fatalf("NewGPIOListenerWithBackend() error = %v", err)
	}

	times := []time.Time{
		time.Unix(0, 0),
		time.Unix(0, int64(5*time.Millisecond)),
		time.Unix(0, int64(100*time.Millisecond)),
	}
	var mu sync.Mutex
	callIndex := 0
	listener.clock = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		current := times[callIndex]
		if callIndex < len(times)-1 {
			callIndex++
		}
		return current
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		listener.Start(ctx)
		close(done)
	}()

	button.changes <- true
	button.changes <- false
	button.changes <- true
	button.changes <- false

	deadline := time.After(time.Second)
	for {
		if len(handler.snapshot()) >= 2 {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("timed out waiting for debounced GPIO events")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()
	<-done

	events := handler.snapshot()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Phase != PhasePress {
		t.Fatalf("first phase = %q, want %q", events[0].Phase, PhasePress)
	}
	if events[1].Phase != PhaseRelease {
		t.Fatalf("second phase = %q, want %q", events[1].Phase, PhaseRelease)
	}
}
