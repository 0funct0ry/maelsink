package events

import (
	"encoding/json"
	"testing"
)

func TestSessionStarted_Payload(t *testing.T) {
	ev := SessionStarted("sess1", "10.0.0.1", "2026-01-01T12:00:00Z")
	if ev.Type != TypeSessionStarted {
		t.Fatalf("Type = %q, want %q", ev.Type, TypeSessionStarted)
	}
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["id"] != "sess1" || got["client_ip"] != "10.0.0.1" || got["started_at"] != "2026-01-01T12:00:00Z" {
		t.Fatalf("payload = %v", got)
	}
}

func TestSessionCompleted_PayloadNullMessageID(t *testing.T) {
	ev := SessionCompleted("sess1", "aborted", nil)
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if v, ok := got["message_id"]; !ok || v != nil {
		t.Fatalf("expected message_id present and null, got %v (present: %v)", v, ok)
	}
	if got["status"] != "aborted" {
		t.Fatalf("status = %v, want aborted", got["status"])
	}
}

func TestSessionLine_Payload(t *testing.T) {
	ev := SessionLine("sess1", 'C', "EHLO client.example.com", 1)
	if ev.Type != TypeSessionLine {
		t.Fatalf("Type = %q, want %q", ev.Type, TypeSessionLine)
	}
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["session_id"] != "sess1" || got["direction"] != "C" || got["line"] != "EHLO client.example.com" || got["position"] != float64(1) {
		t.Fatalf("payload = %v", got)
	}
}

func TestSessionCompleted_PayloadIncludesMessageID(t *testing.T) {
	msgID := "msg1"
	ev := SessionCompleted("sess1", "completed", &msgID)
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["message_id"] != "msg1" {
		t.Fatalf("message_id = %v, want msg1", got["message_id"])
	}
}
