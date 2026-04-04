package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
	"github.com/The-Mess-NZ/gui-punk/pkg/ipc"
)

const defaultSocketPath = "/tmp/guipunk.sock"
const configFileName = "config.json"

type demoState struct {
	colorFlip bool
}

func main() {
	socketPath := flag.String("socket", defaultSocketPath, "Path to the Gooey Unix domain socket")
	autoPatch := flag.Bool("auto-patch", true, "Patch the scene in response to button activation events")
	flag.Parse()

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
	}

	conn, err := net.Dial("unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to connect to Gooey at %s: %v", *socketPath, err)
	}
	defer conn.Close()

	if err := sendCommand(conn, ipc.ActionSubmitScene, buildDemoScene()); err != nil {
		log.Fatalf("Failed to submit demo scene: %v", err)
	}

	log.Printf("Submitted demo scene to %s", *socketPath)
	log.Println("Touch the on-screen buttons to verify touch output and component patching. Press Ctrl+C to exit.")

	eventCh := make(chan ipc.Event)
	errCh := make(chan error, 1)
	go readEvents(conn, eventCh, errCh)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	state := &demoState{}
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				log.Println("Event stream closed")
				return
			}
			if err := handleEvent(conn, event, state, *autoPatch); err != nil {
				log.Printf("Event handling error: %v", err)
			}

		case err := <-errCh:
			if err != nil {
				log.Fatalf("Event stream error: %v", err)
			}
			return

		case <-stop:
			log.Println("Stopping scene demo client")
			return
		}
	}
}

func buildDemoScene() components.SceneDocument {
	return components.SceneDocument{
		Version: components.SceneVersionAlpha1,
		Root: components.SceneNode{
			ID:   "root",
			Type: components.NodeTypeContainer,
			Layout: &components.Layout{
				Direction: components.LayoutDirectionVertical,
				Gap:       8,
				Padding: components.Insets{
					Top:    8,
					Right:  8,
					Bottom: 8,
					Left:   8,
				},
			},
			Style: &components.Style{Background: "#0F151BFF"},
			Children: []components.SceneNode{
				{
					ID:     "title",
					Type:   components.NodeTypeLabel,
					Text:   "Gooey Alpha Demo",
					Bounds: &components.Rect{Height: 26},
					Style: &components.Style{
						Background:  "#1B2732FF",
						Foreground:  "#F7F1D5",
						FontSize:    16,
						TextPadding: 4,
					},
				},
				{
					ID:     "panel",
					Type:   components.NodeTypeContainer,
					Bounds: &components.Rect{Height: 94},
					Layout: &components.Layout{
						Direction: components.LayoutDirectionVertical,
						Gap:       6,
						Padding: components.Insets{
							Top:    8,
							Right:  8,
							Bottom: 8,
							Left:   8,
						},
					},
					Style: &components.Style{
						Background:  "#18222CFF",
						BorderColor: "#3A5568FF",
						BorderWidth: 2,
					},
					Children: []components.SceneNode{
						{
							ID:     "status",
							Type:   components.NodeTypeLabel,
							Text:   "Scene submitted. Touch a button.",
							Bounds: &components.Rect{Height: 28},
							Style: &components.Style{
								Background: "#23313DFF",
								Foreground: "#F2F2E9",
								FontSize:   13,
							},
						},
						{
							ID:     "instructions",
							Type:   components.NodeTypeLabel,
							Text:   "Expected events: touch_press, touch_release, component_activate.",
							Bounds: &components.Rect{Height: 40},
							Style: &components.Style{
								Foreground: "#C7D2DB",
								FontSize:   12,
							},
						},
					},
				},
				{
					ID:     "button-row",
					Type:   components.NodeTypeContainer,
					Bounds: &components.Rect{Height: 56},
					Layout: &components.Layout{
						Direction: components.LayoutDirectionHorizontal,
						Gap:       8,
					},
					Children: []components.SceneNode{
						{
							ID:     "ping-button",
							Type:   components.NodeTypeButton,
							Text:   "Ping",
							Action: "ping",
						},
						{
							ID:     "color-button",
							Type:   components.NodeTypeButton,
							Text:   "Change Status",
							Action: "change_status",
							Style: &components.Style{
								Background:  "#6B3F2FFF",
								Foreground:  "#FFF0E0",
								BorderColor: "#E7B596FF",
								BorderWidth: 2,
							},
						},
					},
				},
				{
					ID:     "footer",
					Type:   components.NodeTypeLabel,
					Text:   "Touch both buttons to test live patches.",
					Bounds: &components.Rect{Height: 20},
					Style: &components.Style{
						Foreground: "#96A6B3",
						FontSize:   11,
					},
				},
			},
		},
	}
}

func handleEvent(conn net.Conn, event ipc.Event, state *demoState, autoPatch bool) error {
	switch event.Type {
	case ipc.EventSceneLoaded:
		var ack ipc.AckPayload
		if err := json.Unmarshal(event.Payload, &ack); err != nil {
			return fmt.Errorf("unmarshal scene_loaded payload: %w", err)
		}
		log.Printf("Scene loaded: root=%s version=%s", ack.ID, ack.Version)
		return nil

	case ipc.EventComponentPatched:
		var ack ipc.AckPayload
		if err := json.Unmarshal(event.Payload, &ack); err != nil {
			return fmt.Errorf("unmarshal component_patched payload: %w", err)
		}
		log.Printf("Component patched: %s", ack.ID)
		return nil

	case ipc.EventProtocolError:
		var protocolErr ipc.ErrorPayload
		if err := json.Unmarshal(event.Payload, &protocolErr); err != nil {
			return fmt.Errorf("unmarshal protocol_error payload: %w", err)
		}
		return fmt.Errorf("protocol error for %s: %s", protocolErr.Action, protocolErr.Message)

	case ipc.EventTouchPress, ipc.EventTouchRelease:
		interaction, err := decodeInteraction(event.Payload)
		if err != nil {
			return err
		}
		log.Printf("%s: component=%s x=%d y=%d", event.Type, interaction.ComponentID, interaction.X, interaction.Y)
		return nil

	case ipc.EventComponentActivate:
		interaction, err := decodeInteraction(event.Payload)
		if err != nil {
			return err
		}
		log.Printf("component_activate: component=%s action=%s x=%d y=%d", interaction.ComponentID, interaction.Action, interaction.X, interaction.Y)
		if !autoPatch {
			return nil
		}
		return patchStatus(conn, state, interaction)

	default:
		log.Printf("Unhandled event %s payload=%s", event.Type, string(event.Payload))
		return nil
	}
}

func patchStatus(conn net.Conn, state *demoState, interaction components.Interaction) error {
	state.colorFlip = !state.colorFlip

	statusText := fmt.Sprintf("Last action: %s at (%d, %d)", interaction.Action, interaction.X, interaction.Y)
	statusStyle := &components.Style{
		Background: "#214357FF",
		Foreground: "#F2F2E9",
	}
	footerText := "Touch both buttons to verify event emission and patching are live."

	if interaction.Action == "change_status" {
		statusText = fmt.Sprintf("Status color toggled by %s", interaction.ComponentID)
		if state.colorFlip {
			statusStyle.Background = "#6D3648FF"
			footerText = "The status label patch is alternating colors from the client."
		} else {
			statusStyle.Background = "#35586AFF"
			footerText = "Press the Change Status button again to flip the patch."
		}
	}

	if err := sendCommand(conn, ipc.ActionPatchComponent, components.ComponentPatch{
		ID:    "status",
		Text:  &statusText,
		Style: statusStyle,
	}); err != nil {
		return fmt.Errorf("patch status label: %w", err)
	}

	return sendCommand(conn, ipc.ActionPatchComponent, components.ComponentPatch{
		ID:   "footer",
		Text: &footerText,
	})
}

func decodeInteraction(payload []byte) (components.Interaction, error) {
	var interaction components.Interaction
	if err := json.Unmarshal(payload, &interaction); err != nil {
		return components.Interaction{}, fmt.Errorf("unmarshal interaction payload: %w", err)
	}
	return interaction, nil
}

func readEvents(conn net.Conn, eventCh chan<- ipc.Event, errCh chan<- error) {
	defer close(eventCh)

	scanner := bufio.NewScanner(conn)
	buffer := make([]byte, 0, 4096)
	scanner.Buffer(buffer, 1024*1024)

	for scanner.Scan() {
		var event ipc.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			errCh <- fmt.Errorf("unmarshal event envelope: %w", err)
			return
		}
		eventCh <- event
	}

	if err := scanner.Err(); err != nil {
		errCh <- err
		return
	}

	errCh <- nil
}

func sendCommand(conn net.Conn, action string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", action, err)
	}

	commandBytes, err := json.Marshal(ipc.Command{
		Action:  action,
		Payload: payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("marshal %s command: %w", action, err)
	}

	commandBytes = append(commandBytes, '\n')
	if _, err := conn.Write(commandBytes); err != nil {
		return fmt.Errorf("write %s command: %w", action, err)
	}

	return nil
}
