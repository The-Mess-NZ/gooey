package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
	"github.com/The-Mess-NZ/gui-punk/pkg/input"
	"github.com/The-Mess-NZ/gui-punk/pkg/ipc"
	"github.com/The-Mess-NZ/gui-punk/pkg/render"
)

const udsSocketPath = "/tmp/guipunk.sock" // For production Midipunk, use /run/guipunk.sock
const fbDevicePath = "/dev/fb0"           // The Raspberry Pi SPI framebuffer device
const configFileName = "config.json"

func main() {
	log.Println("Initializing GUIPunk Framebuffer service...")
	configPath, err := components.LoadConfigFromCandidates(
		os.Getenv("GUIPUNK_CONFIG_PATH"),
		filepath.Join(".", configFileName),
		filepath.Join("..", configFileName),
		filepath.Join("..", "..", configFileName),
		filepath.Join("/etc", "guipunk", configFileName),
	)
	if err != nil {
		log.Fatalf("Failed to load component config: %v", err)
	}
	if configPath != "" {
		log.Printf("Loaded component config from %s", configPath)
	} else {
		log.Println("No component config found; using built-in defaults")
	}
	touchCfg := components.TouchSettings()

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
	touchListener, err := input.NewTouchListener(touchCfg.DevicePath, engine)
	if err != nil {
		log.Printf("Warning: Failed to initialize touch device on %s: %v", touchCfg.DevicePath, err)
		log.Println("GUIPunk will continue without touch interaction support.")
	} else {
		touchListener.MinXRaw = touchCfg.MinXRaw
		touchListener.MaxXRaw = touchCfg.MaxXRaw
		touchListener.MinYRaw = touchCfg.MinYRaw
		touchListener.MaxYRaw = touchCfg.MaxYRaw
		touchListener.ScreenXPixels = touchCfg.ScreenXPixels
		touchListener.ScreenYPixels = touchCfg.ScreenYPixels
		touchListener.IsLandscape = touchCfg.IsLandscape
		touchListener.InvertX = touchCfg.InvertX
		touchListener.InvertY = touchCfg.InvertY

		go touchListener.Start(ctx)
	}

	// 3. Setup IPC (Unix Domain Socket Server)
	server := ipc.NewServer(udsSocketPath)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start IPC Server over %s: %v", udsSocketPath, err)
	}
	defer server.Stop()
	engine.SetInteractionHandler(func(interaction components.Interaction) {
		emitEvent(server, interaction.Kind, interaction)
	})

	// 2. Main Event Loop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Ready. Waiting for host to connect via UDS...")

	for {
		select {
		case cmd := <-server.Commands:
			if err := handleCommand(engine, server, cmd); err != nil {
				log.Printf("Command %s failed: %v", cmd.Action, err)
				emitEvent(server, ipc.EventProtocolError, ipc.ErrorPayload{Action: cmd.Action, Message: err.Error()})
			}

		case <-stop:
			log.Println("Shutting down gracefully...")
			cancel() // Signal render engine loop to exit cleanly
			return
		}
	}
}

func handleCommand(engine *render.Engine, server *ipc.Server, cmd ipc.Command) error {
	switch cmd.Action {
	case ipc.ActionSubmitScene, ipc.ActionReplaceScene:
		var doc components.SceneDocument
		if err := json.Unmarshal(cmd.Payload, &doc); err != nil {
			return fmt.Errorf("invalid scene payload: %w", err)
		}
		if err := engine.LoadScene(doc); err != nil {
			return err
		}
		emitEvent(server, ipc.EventSceneLoaded, ipc.AckPayload{ID: doc.Root.ID, Version: doc.Version})
		return nil

	case ipc.ActionPatchComponent:
		var patch components.ComponentPatch
		if err := json.Unmarshal(cmd.Payload, &patch); err != nil {
			return fmt.Errorf("invalid patch payload: %w", err)
		}
		if err := engine.PatchComponent(patch); err != nil {
			return err
		}
		emitEvent(server, ipc.EventComponentPatched, ipc.AckPayload{ID: patch.ID})
		return nil

	default:
		return fmt.Errorf("unsupported action %q", cmd.Action)
	}
}

// TODO: any, really?
func emitEvent(server *ipc.Server, eventType string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal %s payload: %v", eventType, err)
		return
	}

	event := ipc.Event{Type: eventType, Payload: b}
	select {
	case server.Events <- event:
	default:
		log.Printf("Dropping IPC event %s because the queue is full", eventType)
	}
}
