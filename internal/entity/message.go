package entity

import (
	"fmt"
	"time"
)

// MessageType represents the type of a message
type MessageType string

const (
	// MessageTypeChat represents a regular chat message
	MessageTypeChat MessageType = "chat"

	// MessageTypeSystem represents a system message
	MessageTypeSystem MessageType = "system"

	// MessageTypeNotification represents a notification message
	MessageTypeNotification MessageType = "notification"

	// MessageTypeCommand represents a command message
	MessageTypeCommand MessageType = "command"
)

// Message represents a communication between entities
type Message struct {
	ID        string      // Unique identifier
	Type      MessageType // Type of message
	Content   string      // Message content
	SenderID  string      // ID of the sending entity
	TargetID  string      // ID of the target entity (if direct message) or chat
	CreatedAt time.Time   // When the message was created
	Metadata  Metadata    // Additional metadata
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, content string, senderID, targetID string) Message {
	return Message{
		ID:        GenerateID(),
		Type:      msgType,
		Content:   content,
		SenderID:  senderID,
		TargetID:  targetID,
		CreatedAt: time.Now(),
		Metadata:  make(Metadata),
	}
}

// GenerateID generates a unique ID for a message
func GenerateID() string {
	// Simple implementation for now
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
}
