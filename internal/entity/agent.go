package entity

import (
	"goproduct/internal/memory"
)

// Agent extends the base Entity interface with AI agent-specific capabilities
type Agent interface {
	Entity
	ToolUser
	MemoryAccess

	// Agent-specific methods
	IsActive() bool              // Whether the agent is currently active
	SetActive(active bool) error // Activate or deactivate the agent

	// Capability management
	Capabilities() []string         // List of capabilities this agent has
	HasCapability(cap string) bool  // Check if agent has a specific capability
	AddCapability(cap string) error // Add a capability to this agent

	// Memory-specific methods (extending MemoryAccess)
	PersonalMemories() ([]memory.MemoryRecord, error) // Get only memories owned by this agent
	SharedMemories() ([]memory.MemoryRecord, error)   // Get memories shared with this agent

	// Task management
	CurrentTasks() []string           // Get IDs of tasks currently assigned to this agent
	AssignTask(taskID string) error   // Assign a new task to this agent
	CompleteTask(taskID string) error // Mark a task as completed

	// Agent statistics
	TotalInteractions() int // Total number of interactions handled
	SuccessRate() float64   // Success rate of tasks/interactions
	LastActiveTime() string // When the agent was last active
}
