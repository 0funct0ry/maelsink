package ws

import (
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/version"
)

// frame is the JSON text-frame shape sent to every client, per SPEC.md §5.5:
// {"type": "...", "payload": {...}}.
type frame struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// helloFrame builds the frame sent immediately on connect (SPEC.md §5.5).
func helloFrame() frame {
	return frame{Type: "hello", Payload: map[string]string{
		"server_time": time.Now().UTC().Format(time.RFC3339),
		"version":     version.Version,
	}}
}

// shutdownFrame builds the frame broadcast on graceful shutdown.
func shutdownFrame() frame {
	return frame{Type: "server.shutdown", Payload: struct{}{}}
}

// eventFrame builds the frame forwarding a bus event to WS clients.
func eventFrame(ev events.Event) frame {
	return frame{Type: string(ev.Type), Payload: ev.Payload}
}
