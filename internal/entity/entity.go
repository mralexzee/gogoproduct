package entity

import (
	"goproduct/internal/messaging"
	"time"
)

// EntityType represents the type of an entity in the system
type EntityType string

const (
	// EntityTypeHuman represents a human entity
	EntityTypeHuman EntityType = "human"

	// EntityTypeAgent represents an AI agent entity
	EntityTypeAgent EntityType = "agent"

	// EntityTypeSystem represents a system entity
	EntityTypeSystem EntityType = "system"

	// EntityTypeGroup represents a group entity
	EntityTypeGroup EntityType = "group"
)

// EntityStatus represents the current status of an entity
type EntityStatus string

const (
	// StatusActive indicates the entity is active and available
	StatusActive EntityStatus = "active"

	// StatusInactive indicates the entity is temporarily inactive
	StatusInactive EntityStatus = "inactive"

	// StatusBusy indicates the entity is currently busy
	StatusBusy EntityStatus = "busy"
)

// Role represents a functional role an entity can have
type Role string

const (
	// RoleAdmin represents administrative capabilities
	RoleAdmin Role = "admin"

	// RoleUser represents standard user capabilities
	RoleUser Role = "user"

	// RoleDeveloper represents developer capabilities
	RoleDeveloper Role = "developer"

	// RoleManager represents manager capabilities
	RoleManager Role = "manager"
)

// Metadata contains additional information about an entity
type Metadata map[string]interface{}

// Entity represents any actor in the system, human or AI
type Entity interface {
	// Core identity methods
	ID() string
	Name() string
	Type() EntityType

	// Status management
	Status() EntityStatus
	SetStatus(status EntityStatus) error

	// Profile and metadata
	Metadata() Metadata
	SetMetadata(key string, value interface{}) error

	// Role management
	Roles() []Role
	HasRole(role Role) bool
	AddRole(role Role) error
	RemoveRole(role Role) error

	// Creation/modification info
	CreatedAt() time.Time
	UpdatedAt() time.Time

	// Communication methods
	CanReceiveMessage() bool // Whether this entity can receive messages
	CanSendMessage() bool    // Whether this entity can send messages

	// Messaging methods
	ReceiveMessage(msg messaging.Message) error                                                     // Handle an incoming message
	SendMessage(recipients []string, contentType string, content []byte) (messaging.Message, error) // Send a message
}
