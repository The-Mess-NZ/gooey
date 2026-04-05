package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/The-Mess-NZ/gui-punk/pkg/components"
)

// Client is a small host-side helper for talking to Gooey over the UDS protocol.
type Client struct {
	conn    net.Conn
	reader  *bufio.Scanner
	writeMu sync.Mutex
}

// Dial connects to a running Gooey instance over a Unix domain socket.
func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	return &Client{conn: conn, reader: scanner}, nil
}

// Close closes the underlying socket connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// SubmitScene sends a full scene document to Gooey.
func (c *Client) SubmitScene(scene components.SceneDocument) error {
	return c.SendCommand(ActionSubmitScene, scene)
}

// ReplaceScene replaces the current scene document in Gooey.
func (c *Client) ReplaceScene(scene components.SceneDocument) error {
	return c.SendCommand(ActionReplaceScene, scene)
}

// PatchComponent sends a targeted patch for a single scene node.
func (c *Client) PatchComponent(patch components.ComponentPatch) error {
	return c.SendCommand(ActionPatchComponent, patch)
}

// SendCommand sends a generic transport envelope with a JSON payload.
func (c *Client) SendCommand(action string, payload any) error {
	if c == nil || c.conn == nil {
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
	if _, err := c.conn.Write(commandBytes); err != nil {
		return fmt.Errorf("write %s command: %w", action, err)
	}

	return nil
}

// NextEvent blocks until the next event arrives from Gooey.
func (c *Client) NextEvent() (Event, error) {
	if c == nil || c.reader == nil {
		return Event{}, fmt.Errorf("ipc client is not connected")
	}
	if !c.reader.Scan() {
		if err := c.reader.Err(); err != nil {
			return Event{}, err
		}
		return Event{}, fmt.Errorf("ipc connection closed")
	}

	var event Event
	if err := json.Unmarshal(c.reader.Bytes(), &event); err != nil {
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
