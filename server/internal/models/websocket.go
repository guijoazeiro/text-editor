package models

import "github.com/google/uuid"

type MessageType string

const (
	MessageTypeJoin         MessageType = "join"
	MessageTypeLeave        MessageType = "leave"
	MessageTypeEdit         MessageType = "edit"
	MessageTypeCursor       MessageType = "cursor"
	MessageTypePresence     MessageType = "presence"
	MessageTypeSync         MessageType = "sync"
	MessageTypeAwareness    MessageType = "awareness"
	MessageTypeYjsSync      MessageType = "yjs-sync"
	MessageTypeYjsAwareness    MessageType = "yjs-awareness"
	MessageTypeYjsInit         MessageType = "yjs-init"
	MessageTypeYjsAwarenessOff    MessageType = "yjs-awareness-off"
	MessageTypeYjsReset           MessageType = "yjs-reset"
	MessageTypeDocumentContentReset MessageType = "document-content-reset"
)

type WSMessage struct {
	Type       MessageType            `json:"type"`
	DocumentID string                 `json:"document_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	UserName   string                 `json:"user_name,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  int64                  `json:"timestamp,omitempty"`
}

type CursorPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type UserPresence struct {
	UserID   uuid.UUID      `json:"user_id"`
	UserName string         `json:"user_name"`
	Color    string         `json:"color"`
	Cursor   CursorPosition `json:"cursor"`
	Online   bool           `json:"online"`
}
