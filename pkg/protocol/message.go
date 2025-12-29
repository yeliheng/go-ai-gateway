package protocol

import "encoding/json"

// MessageType defines the type of message.
type MessageType string

const (
	TypeChat   MessageType = "chat"
	TypePing   MessageType = "ping"
	TypePong   MessageType = "pong"
	TypeError  MessageType = "error"
	TypeSystem MessageType = "system"
)

// Message is the standard envelope for all WebSocket communication.
type Message struct {
	Type     MessageType     `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// ChatPayload represents the content for a chat message.
type ChatPayload struct {
	Content string `json:"content"`
	Type    string `json:"type,omitempty"` // "text" or "reasoning"
	Model   string `json:"model,omitempty"`
}

// ErrorPayload represents an error message.
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
