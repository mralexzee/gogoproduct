package messaging

import (
	"goproduct/internal/tracing"
)

// MessageHandler processes incoming messages
type MessageHandler func(Message) error

// MessageBus handles routing of messages between entities
type MessageBus interface {
	// Publish a message to its recipients
	Publish(message Message) error

	// Subscribe to receive messages for an entity
	Subscribe(entityID string, handler MessageHandler) error

	// Unsubscribe entity from receiving messages
	Unsubscribe(entityID string) error

	// Group management
	CreateGroup(groupID, name string, members []string) error
	AddToGroup(groupID, entityID string) error
	RemoveFromGroup(groupID, entityID string) error
	GetGroupMembers(groupID string) ([]string, error)

	// Tracer management
	SetTracer(tracer tracing.Tracer)
	GetTracer() tracing.Tracer
}
