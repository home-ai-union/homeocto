package home

import "time"

// Message type constants (Pico Protocol compatible).
const (
	TypeMessageSend   = "message.send"
	TypeMessageCreate = "message.create"
	TypeMediaSend     = "media.send"
	TypeError         = "error"
	TypePing          = "ping"
	TypePong          = "pong"
)

// homeMessage represents a Pico Protocol compatible message.
type homeMessage struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

func newMessage(msgType string, payload map[string]any) homeMessage {
	return homeMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

func newError(code, message string) homeMessage {
	return newMessage(TypeError, map[string]any{
		"code":    code,
		"message": message,
	})
}
