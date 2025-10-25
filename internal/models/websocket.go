package models

import "github.com/google/uuid"

// WebSocket event types
const (
	EventMessageNew     = "message.new"
	EventMessageSend    = "message.send"
	EventMessageRead    = "message.read"
	EventTypingStart    = "typing.start"
	EventTypingStop     = "typing.stop"
	EventPresenceUpdate = "presence.update"
	EventError          = "error"
)

type WSMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type WSMessageSendPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Body           string    `json:"body"`
}

type WSMessageReadPayload struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
}

type WSTypingPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}

type WSErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
