package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/The-Mess-NZ/gui-punk/pkg/input"
	"github.com/The-Mess-NZ/gui-punk/pkg/ipc"
	"github.com/The-Mess-NZ/gui-punk/pkg/render"
)

const udsSocketPath = "/tmp/guipunk.sock" // For production Midipunk, use /run/guipunk.sock
const fbDevicePath = "/dev/fb0"           // The Raspberry Pi SPI framebuffer device
const touchDevPath = "/dev/input/event0"  // Resistive touch STMPE-TS Evdev controller

func main() {
	log.Println("Initializing GUIPunk Framebuffer service...")

	// 1. Setup Graphics Rendering Engine
	engine, err := render.NewEngine(fbDevicePath)
	if err != nil {
		log.Fatalf("Failed to initialize Render Engine on %s: %v", fbDevicePath, err)
	}
	defer engine.Close()

	// Create context to manage background loop lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spin up the event-driven render loop (CPU stays near 0% when idle)
	go engine.Loop(ctx)

	// 2. Setup Touch Input via Evdev listener
	// TODO: The touch functionality and the rendering feel too tightly coupled right now.
	touchListener, err := input.NewTouchListener(touchDevPath, engine)
	if err != nil {
		log.Printf("Warning: Failed to initialize touch device on %s: %v", touchDevPath, err)
		log.Println("GUIPunk will continue without touch interaction support.")
	} else {
		// Example defaults, Midipunk likely needs tuning here
		touchListener.SwapXY = true  // Very common for generic TFTs to swap X/Y axes internally
		touchListener.InvertX = true // Tweak to align

		go touchListener.Start(ctx)
	}

	// 3. Setup IPC (Unix Domain Socket Server)
	server := ipc.NewServer(udsSocketPath)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start IPC Server over %s: %v", udsSocketPath, err)
	}
	defer server.Stop()

	// 2. Main Event Loop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Ready. Waiting for host to connect via UDS...")

	for {
		select {
		case cmd := <-server.Commands:
			log.Printf("Received Command: %s, Payload: %s\n", cmd.Action, string(cmd.Payload))
			// trigger redraw whenever state changes for now
			engine.TriggerRedraw()

		case evt := <-server.Events:
			log.Printf("Dispatching Event to host: %s, Payload: %s\n", evt.Type, string(evt.Payload))
			// Just an echo chamber for now mapping out

		case <-stop:
			log.Println("Shutting down gracefully...")
			cancel() // Signal render engine loop to exit cleanly
			return
		}
	}
}
