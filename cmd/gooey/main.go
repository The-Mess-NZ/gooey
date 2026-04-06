package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/The-Mess-NZ/gooey/pkg/components"
	"github.com/The-Mess-NZ/gooey/pkg/input"
	"github.com/The-Mess-NZ/gooey/pkg/ipc"
	"github.com/The-Mess-NZ/gooey/pkg/render"
)

const udsSocketPath = "/tmp/gooey.sock" // For production Midipunk, use /run/gooey.sock
const fbDevicePath = "/dev/fb0"         // The Raspberry Pi SPI framebuffer device
const configFileName = "config.json"

func main() {
	log.Println("Initializing Gooey Framebuffer service...")
	configPath, err := components.LoadConfigFromCandidates(
		os.Getenv("GOOEY_CONFIG_PATH"),
		filepath.Join(".", configFileName),
		filepath.Join("..", configFileName),
		filepath.Join("..", "..", configFileName),
		filepath.Join("/etc", "gooey", configFileName),
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
	gpioCfg := components.GPIOSettings()

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
		log.Println("Gooey will continue without touch interaction support.")
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

	// 3. Setup GPIO Input listener
	gpioSpecs, err := buildGPIOButtonSpecs(gpioCfg)
	if err != nil {
		log.Printf("Warning: Invalid GPIO configuration: %v", err)
		log.Println("Gooey will continue without GPIO interaction support.")
	} else if len(gpioSpecs) > 0 {
		gpioListener, err := input.NewGPIOListener(gpioSpecs, engine)
		if err != nil {
			log.Printf("Warning: Failed to initialize GPIO listener: %v", err)
			log.Println("Gooey will continue without GPIO interaction support.")
		} else {
			go gpioListener.Start(ctx)
		}
	}

	// 4. Setup IPC (Unix Domain Socket Server)
	server := ipc.NewServer(udsSocketPath)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start IPC Server over %s: %v", udsSocketPath, err)
	}
	defer server.Stop()
	engine.SetEventHandler(func(eventType string, payload any) {
		emitEvent(server, eventType, payload)
	})

	// 2. Main Event Loop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Ready. Waiting for host to connect via UDS...")

	for {
		select {
		case cmd := <-server.Commands:
			if err := handleCommand(engine, server, cmd, configPath); err != nil {
				payload := protocolErrorPayload(cmd.Action, err)
				log.Printf("Command %s failed [%s]: %v", cmd.Action, payload.Code, err)
				emitEvent(server, ipc.EventProtocolError, payload)
			}

		case <-stop:
			log.Println("Shutting down gracefully...")
			cancel() // Signal render engine loop to exit cleanly
			return
		}
	}
}

func handleCommand(engine *render.Engine, server *ipc.Server, cmd ipc.Command, configPath string) error {
	switch cmd.Action {
	case ipc.ActionSubmitScene, ipc.ActionReplaceScene:
		var doc components.SceneDocument
		// First try to unmarshal the raw payload into a SceneDocument
		if err := json.Unmarshal(cmd.Payload, &doc); err != nil {
			return &components.ValidationError{Code: components.ValidationCodeInvalidScene, Message: fmt.Sprintf("invalid scene payload: %v", err)}
		}
		// Then validate the parsed document for semantic correctness
		if err := components.ValidateSceneDocument(doc); err != nil {
			return err
		}
		if doc.Version == "" {
			doc.Version = components.SceneVersionAlpha1
		}
		if err := engine.LoadScene(doc); err != nil {
			return err
		}
		emitEvent(server, ipc.EventSceneLoaded, ipc.AckPayload{ID: doc.Root.ID, Version: doc.Version})
		return nil

	case ipc.ActionPatchComponent:
		var patch components.ComponentPatch
		// First try to unmarshal the raw payload into a ComponentPatch
		if err := json.Unmarshal(cmd.Payload, &patch); err != nil {
			return &components.ValidationError{Code: components.ValidationCodeInvalidPatch, Message: fmt.Sprintf("invalid patch payload: %v", err)}
		}
		// Then validate the parsed patch for semantic correctness
		if err := components.ValidateComponentPatch(patch); err != nil {
			return err
		}
		if err := engine.PatchComponent(patch); err != nil {
			return err
		}
		emitEvent(server, ipc.EventComponentPatched, ipc.AckPayload{ID: patch.ID})
		return nil

	case ipc.ActionGetStatus:
		emitEvent(server, ipc.EventDiagnostics, runtimeDiagnostics(engine, configPath))
		return nil

	default:
		return fmt.Errorf("unsupported action %q", cmd.Action)
	}
}

func runtimeDiagnostics(engine *render.Engine, configPath string) ipc.DiagnosticsPayload {
	touchCfg := components.TouchSettings()
	gpioCfg := components.GPIOSettings()
	snapshot := engine.Snapshot()
	return ipc.DiagnosticsPayload{
		Kind:    "status_snapshot",
		Message: "current Gooey runtime status",
		Status: &ipc.StatusPayload{
			SceneLoaded:     snapshot.SceneLoaded,
			SceneVersion:    snapshot.SceneVersion,
			RootID:          snapshot.RootID,
			ComponentCount:  snapshot.ComponentCount,
			TouchConfigured: touchCfg.DevicePath != "" && touchCfg.DevicePath != "/dev/null",
			TouchDevicePath: touchCfg.DevicePath,
			GPIOConfigured:  len(gpioCfg.Buttons) > 0,
			GPIOInputCount:  len(gpioCfg.Buttons),
			ConfigPath:      configPath,
		},
	}
}

func buildGPIOButtonSpecs(cfg components.GPIOConfig) ([]input.GPIOButtonSpec, error) {
	if len(cfg.Buttons) == 0 {
		return nil, nil
	}

	specs := make([]input.GPIOButtonSpec, 0, len(cfg.Buttons))
	for _, button := range cfg.Buttons {
		emitPress, emitRelease, err := input.EmitPhases(button.Edges)
		if err != nil {
			return nil, fmt.Errorf("gpio button %q: %w", button.ID, err)
		}
		specs = append(specs, input.GPIOButtonSpec{
			ID:          button.ID,
			Pin:         button.Pin.String(),
			Pull:        button.Pull,
			Invert:      button.Invert,
			Debounce:    time.Duration(button.DebounceMs) * time.Millisecond,
			EmitPress:   emitPress,
			EmitRelease: emitRelease,
		})
	}
	return specs, nil
}

// TODO: Is this in the right place?
func protocolErrorPayload(action string, err error) ipc.ErrorPayload {
	payload := ipc.ErrorPayload{Action: action, Code: ipc.ErrorCodeInternal, Message: err.Error()}
	var validationErr *components.ValidationError
	if errors.As(err, &validationErr) {
		switch validationErr.Code {
		case components.ValidationCodeInvalidScene:
			payload.Code = ipc.ErrorCodeInvalidScene
		case components.ValidationCodeInvalidPatch:
			payload.Code = ipc.ErrorCodeInvalidPatch
		default:
			payload.Code = validationErr.Code
		}
		payload.Message = validationErr.Message
		return payload
	}

	if err.Error() == "scene is not loaded" {
		payload.Code = ipc.ErrorCodeSceneNotLoaded
		return payload
	}
	if len(action) > 0 && len(err.Error()) >= len("unsupported action") && err.Error()[:len("unsupported action")] == "unsupported action" {
		payload.Code = ipc.ErrorCodeUnknownAction
	}
	return payload
}

// TODO: any, really? Can we type better?
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
