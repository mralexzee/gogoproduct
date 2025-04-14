package entity

import (
	"fmt"
	"goproduct/internal/messaging"
	"time"

	"github.com/google/uuid"
)

// Group extends the base Entity interface with group-specific capabilities
type Group interface {
	Entity

	// Group-specific methods
	AddMember(memberID string) error
	RemoveMember(memberID string) error
	Members() []string
	IsMember(entityID string) bool
}

// BasicGroup provides a standard implementation of the Group interface
type BasicGroup struct {
	id         string
	name       string
	entityType EntityType
	status     EntityStatus
	members    map[string]bool
	metadata   Metadata
	roles      map[Role]bool
	createdAt  string
	updatedAt  string
}

// NewGroup creates a new basic group
func NewGroup(id, name string) *BasicGroup {
	// If ID is not provided, generate a new UUID
	if id == "" {
		id = uuid.New().String()
	}

	now := time.Now().Format(time.RFC3339)
	return &BasicGroup{
		id:         id,
		name:       name,
		entityType: EntityTypeGroup,
		status:     StatusActive,
		members:    make(map[string]bool),
		metadata:   make(Metadata),
		roles:      make(map[Role]bool),
		createdAt:  now,
		updatedAt:  now,
	}
}

// Core identity methods
func (g *BasicGroup) ID() string {
	return g.id
}

func (g *BasicGroup) Name() string {
	return g.name
}

func (g *BasicGroup) Type() EntityType {
	return g.entityType
}

// Status management
func (g *BasicGroup) Status() EntityStatus {
	return g.status
}

func (g *BasicGroup) SetStatus(status EntityStatus) error {
	g.status = status
	return nil
}

// Profile and metadata
func (g *BasicGroup) Metadata() Metadata {
	return g.metadata
}

func (g *BasicGroup) SetMetadata(key string, value interface{}) error {
	g.metadata[key] = value
	return nil
}

// Role management
func (g *BasicGroup) Roles() []Role {
	roles := make([]Role, 0, len(g.roles))
	for role := range g.roles {
		roles = append(roles, role)
	}
	return roles
}

func (g *BasicGroup) HasRole(role Role) bool {
	_, has := g.roles[role]
	return has
}

func (g *BasicGroup) AddRole(role Role) error {
	g.roles[role] = true
	return nil
}

func (g *BasicGroup) RemoveRole(role Role) error {
	delete(g.roles, role)
	return nil
}

// Creation/modification info
func (g *BasicGroup) CreatedAt() time.Time {
	// Parse the time from the RFC3339 string
	t, _ := time.Parse(time.RFC3339, g.createdAt)
	return t
}

func (g *BasicGroup) UpdatedAt() time.Time {
	// Parse the time from the RFC3339 string
	t, _ := time.Parse(time.RFC3339, g.updatedAt)
	return t
}

// Communication methods
func (g *BasicGroup) CanReceiveMessage() bool {
	return true
}

func (g *BasicGroup) CanSendMessage() bool {
	return false // Groups can't send messages directly
}

// Group members manage messaging through the message bus
func (g *BasicGroup) ReceiveMessage(msg messaging.Message) error {
	// Groups don't process messages directly - they're handled by the message bus
	return nil
}

func (g *BasicGroup) SendMessage(recipients []string, contentType string, content []byte) (messaging.Message, error) {
	// Groups can't send messages
	return messaging.Message{}, fmt.Errorf("groups cannot send messages directly")
}

// Group-specific methods
func (g *BasicGroup) AddMember(memberID string) error {
	g.members[memberID] = true
	return nil
}

func (g *BasicGroup) RemoveMember(memberID string) error {
	delete(g.members, memberID)
	return nil
}

func (g *BasicGroup) Members() []string {
	members := make([]string, 0, len(g.members))
	for memberID := range g.members {
		members = append(members, memberID)
	}
	return members
}

func (g *BasicGroup) IsMember(entityID string) bool {
	_, isMember := g.members[entityID]
	return isMember
}
