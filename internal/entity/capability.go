package entity

import (
	"goproduct/internal/memory"
)

// MessageHandler is a capability for entities that can process messages
type MessageHandler interface {
	// HandleMessage processes an incoming message and optionally returns a response
	HandleMessage(message Message) ([]Message, error)
}

// ToolUser is a capability for entities that can use tools
type ToolUser interface {
	// UseTool uses a tool with the given parameters and returns the result
	UseTool(toolID string, params map[string]interface{}) (interface{}, error)

	// AvailableTools returns the list of tools available to this entity
	AvailableTools() []string
}

// MemoryAccess is a capability for entities that can access the memory system
type MemoryAccess interface {
	// ReadMemory retrieves memories based on a filter
	ReadMemory(filter memory.MemoryFilter) ([]memory.MemoryRecord, error)

	// WriteMemory stores a new memory
	WriteMemory(record memory.MemoryRecord) error
}

// TeamMember is a capability for entities that can be part of teams
type TeamMember interface {
	// Teams returns the teams this entity belongs to
	Teams() []string

	// JoinTeam adds the entity to a team
	JoinTeam(teamID string) error

	// LeaveTeam removes the entity from a team
	LeaveTeam(teamID string) error

	// IsInTeam checks if the entity is part of a team
	IsInTeam(teamID string) bool
}

// ChatParticipant is a capability for entities that can participate in chats
type ChatParticipant interface {
	// JoinChat joins a chat session
	JoinChat(chatID string) error

	// LeaveChat leaves a chat session
	LeaveChat(chatID string) error

	// SendMessage sends a message to a chat
	SendMessage(chatID string, content string) (Message, error)

	// ActiveChats returns the list of active chats
	ActiveChats() []string
}
