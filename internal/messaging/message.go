package messaging

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// BroadcastAddress is used to send a message to all entities
	BroadcastAddress = "*"

	// Content types
	ContentTypeText    = "text/plain"
	ContentTypeJSON    = "application/json"
	ContentTypeCommand = "application/x-command"
)

// Message represents communication between entities
type Message struct {
	ID          string   // UUID for the message
	SenderID    string   // UUID of the sending entity
	Recipients  []string // UUIDs of recipient entities
	ContentType string   // MIME type
	Content     []byte   // Raw binary content
	ReplyToID   string   // UUID of message being replied to
	Timestamp   time.Time
	Metadata    map[string]string
}

// NewMessage creates a new message
func NewMessage(senderID string, recipients []string, contentType string, content []byte) Message {
	return Message{
		ID:          uuid.New().String(),
		SenderID:    senderID,
		Recipients:  recipients,
		ContentType: contentType,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// NewTextMessage creates a plain text message
func NewTextMessage(senderID string, recipients []string, text string) Message {
	return NewMessage(senderID, recipients, ContentTypeText, []byte(text))
}

// NewJSONMessage creates a JSON message
func NewJSONMessage(senderID string, recipients []string, jsonData []byte) Message {
	return NewMessage(senderID, recipients, ContentTypeJSON, jsonData)
}

// NewReplyMessage creates a new message in reply to another message
func NewReplyMessage(senderID string, originalMsg Message, contentType string, content []byte) Message {
	// Create a new message with a unique ID
	msg := NewMessage(senderID, []string{originalMsg.SenderID}, contentType, content)

	// Set the ReplyToID to link this message to the original
	msg.ReplyToID = originalMsg.ID

	// Propagate any relevant metadata
	for k, v := range originalMsg.Metadata {
		if strings.HasPrefix(k, "conversation_") || strings.HasPrefix(k, "thread_") {
			msg.Metadata[k] = v
		}
	}

	return msg
}

// NewTextReplyMessage creates a new text message in reply to another message
func NewTextReplyMessage(senderID string, originalMsg Message, text string) Message {
	return NewReplyMessage(senderID, originalMsg, ContentTypeText, []byte(text))
}

// TextContent extracts the text content of a message
func (m Message) TextContent() (string, error) {
	if m.ContentType != ContentTypeText {
		return "", fmt.Errorf("message content is not text: %s", m.ContentType)
	}
	return string(m.Content), nil
}

// WithReplyTo sets the message as a reply to another message
func (m Message) WithReplyTo(replyToID string) Message {
	m.ReplyToID = replyToID
	return m
}
