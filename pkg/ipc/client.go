package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/The-Mess-NZ/gooey/pkg/components"
)

// Client is a small host-side helper for talking to Gooey over the UDS protocol.
type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Scanner
	mu         sync.Mutex
	writeMu    sync.Mutex
	lastScene  *components.SceneDocument
}

// Dial connects to a running Gooey instance over a Unix domain socket.
func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	return &Client{socketPath: socketPath, conn: conn, reader: scanner}, nil
}

// Close closes the underlying socket connection.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	return err
}

// SubmitScene sends a full scene document to Gooey.
func (c *Client) SubmitScene(scene components.SceneDocument) error {
	if err := c.SendCommand(ActionSubmitScene, scene); err != nil {
		return err
	}
	c.rememberScene(scene)
	return nil
}

// ReplaceScene replaces the current scene document in Gooey.
func (c *Client) ReplaceScene(scene components.SceneDocument) error {
	if err := c.SendCommand(ActionReplaceScene, scene); err != nil {
		return err
	}
	c.rememberScene(scene)
	return nil
}

// PatchComponent sends a targeted patch for a single scene node.
func (c *Client) PatchComponent(patch components.ComponentPatch) error {
	return c.SendCommand(ActionPatchComponent, patch)
}

// RequestStatus asks Gooey to emit a diagnostics event with its current runtime status.
func (c *Client) RequestStatus() error {
	return c.SendCommand(ActionGetStatus, struct{}{})
}

// Reconnect closes any current connection and dials the configured socket again.
func (c *Client) Reconnect() error {
	if c == nil {
		return fmt.Errorf("ipc client is not connected")
	}
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	c.mu.Lock()
	oldConn := c.conn
	c.conn = conn
	c.reader = scanner
	c.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}
	return nil
}

// ReconnectAndResubmit reconnects to Gooey and resubmits the last full scene if one exists.
func (c *Client) ReconnectAndResubmit() error {
	if err := c.Reconnect(); err != nil {
		return err
	}

	if scene, ok := c.RememberedScene(); ok {
		if err := c.SubmitScene(scene); err != nil {
			return err
		}
	}
	return c.RequestStatus()
}

// RememberedScene returns the last successfully submitted full scene, if any.
func (c *Client) RememberedScene() (components.SceneDocument, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastScene == nil {
		return components.SceneDocument{}, false
	}
	return cloneScene(*c.lastScene), true
}

// SendCommand sends a generic transport envelope with a JSON payload.
func (c *Client) SendCommand(action string, payload any) error {
	if c == nil {
		return fmt.Errorf("ipc client is not connected")
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("ipc client is not connected")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", action, err)
	}

	commandBytes, err := json.Marshal(Command{Action: action, Payload: payloadBytes})
	if err != nil {
		return fmt.Errorf("marshal %s command: %w", action, err)
	}
	commandBytes = append(commandBytes, '\n')

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := conn.Write(commandBytes); err != nil {
		return fmt.Errorf("write %s command: %w", action, err)
	}

	return nil
}

// NextEvent blocks until the next event arrives from Gooey.
func (c *Client) NextEvent() (Event, error) {
	if c == nil {
		return Event{}, fmt.Errorf("ipc client is not connected")
	}

	c.mu.Lock()
	reader := c.reader
	c.mu.Unlock()
	if reader == nil {
		return Event{}, fmt.Errorf("ipc client is not connected")
	}
	if !reader.Scan() {
		if err := reader.Err(); err != nil {
			return Event{}, err
		}
		return Event{}, fmt.Errorf("ipc connection closed")
	}

	var event Event
	if err := json.Unmarshal(reader.Bytes(), &event); err != nil {
		return Event{}, fmt.Errorf("unmarshal event envelope: %w", err)
	}
	return event, nil
}

// DecodeEventPayload unmarshals a typed event payload into the caller-provided target.
func DecodeEventPayload(event Event, target any) error {
	if err := json.Unmarshal(event.Payload, target); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", event.Type, err)
	}
	return nil
}

func (c *Client) rememberScene(scene components.SceneDocument) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := cloneScene(scene)
	c.lastScene = &cloned
}

func cloneScene(scene components.SceneDocument) components.SceneDocument {
	b, err := json.Marshal(scene)
	if err != nil {
		return scene
	}
	var cloned components.SceneDocument
	if err := json.Unmarshal(b, &cloned); err != nil {
		return scene
	}
	return cloned
}
