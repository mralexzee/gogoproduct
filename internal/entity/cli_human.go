package entity

import (
	"context"
	"goproduct/internal/logging"
	"goproduct/internal/messaging"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CliHumanEntity represents a human user interacting through the CLI
type CliHumanEntity struct {
	id         string
	name       string
	status     EntityStatus
	createdAt  time.Time
	updatedAt  time.Time
	messageBus messaging.MessageBus
	roles      map[Role]bool
	metadata   Metadata
	mutex      sync.RWMutex
	// Track both specific message handlers and a global message handler
	handlers             map[string]func(msg messaging.Message)
	generalHandler       func(msg messaging.Message) // General message handler for all messages
	conversationMutex    sync.RWMutex
	pendingConversations map[string]time.Time // Track messages we're waiting for responses to
	ctx                  context.Context
	cancel               context.CancelFunc
	logger               *logging.Logger
}

// NewCliHumanEntity creates a new CLI human entity
func NewCliHumanEntity(name string, bus messaging.MessageBus) *CliHumanEntity {
	now := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	logger := logging.Get()

	return &CliHumanEntity{
		id:                   uuid.New().String(),
		name:                 name,
		status:               StatusActive,
		createdAt:            now,
		updatedAt:            now,
		messageBus:           bus,
		roles:                map[Role]bool{RoleUser: true},
		metadata:             make(Metadata),
		handlers:             make(map[string]func(msg messaging.Message)),
		pendingConversations: make(map[string]time.Time),
		ctx:                  ctx,
		cancel:               cancel,
		logger:               logger,
	}
}

// Start initializes the CLI human entity and subscribes to messages
func (c *CliHumanEntity) Start() error {
	// This is handled by the message bus subscription
	c.logger.Debug("CLI human entity starting subscription", "entity_id", c.id, "name", c.name)
	return c.messageBus.Subscribe(c.id, func(msg messaging.Message) error {
		c.mutex.RLock()
		// We need to temporarily store these to avoid locking during handler execution
		var specificHandler func(msg messaging.Message)
		var generalHandler func(msg messaging.Message)
		var hasSpecificHandler bool

		// Check if there's a direct handler for this exact message ID
		specificHandler, hasSpecificHandler = c.handlers[msg.ID]
		generalHandler = c.generalHandler
		c.mutex.RUnlock()

		c.logger.Debug("CLI human received message from bus",
			"entity_id", c.id,
			"message_id", msg.ID,
			"sender", msg.SenderID,
			"content_length", len(msg.Content),
			"has_direct_handler", hasSpecificHandler,
			"has_general_handler", generalHandler != nil)

		// Check if it's a response to a message we're tracking
		c.processMessageAsResponse(msg)

		// Execute specific handler if available
		if hasSpecificHandler {
			c.logger.Debug("Executing specific handler", "message_id", msg.ID)
			specificHandler(msg)

			// Remove the handler after execution
			c.mutex.Lock()
			delete(c.handlers, msg.ID)
			c.mutex.Unlock()
			c.logger.Debug("Specific handler executed and removed", "message_id", msg.ID)
			return nil
		}

		// If no specific handler, try the general handler
		if generalHandler != nil {
			c.logger.Debug("Executing general handler", "message_id", msg.ID)
			generalHandler(msg)
			return nil
		}

		c.logger.Debug("No handler found for message", "message_id", msg.ID)
		return nil
	})
}

// Shutdown stops the CLI human entity
func (c *CliHumanEntity) Shutdown() error {
	c.cancel()
	return c.messageBus.Unsubscribe(c.id)
}

// SetGeneralMessageHandler sets a general handler for all incoming messages
func (c *CliHumanEntity) SetGeneralMessageHandler(handler func(msg messaging.Message)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.generalHandler = handler
	c.logger.Debug("General message handler set", "entity_id", c.id)
}

// RegisterMessageHandler registers a callback for a specific message
func (c *CliHumanEntity) RegisterMessageHandler(messageID string, handler func(msg messaging.Message)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.logger.Debug("Registering message handler", "message_id", messageID, "entity_id", c.id)
	c.handlers[messageID] = handler

	// Also track this as a pending conversation
	c.conversationMutex.Lock()
	defer c.conversationMutex.Unlock()
	c.pendingConversations[messageID] = time.Now()
	c.logger.Debug("Added to pending conversations", "message_id", messageID, "pending_count", len(c.pendingConversations))
}

// Entity interface implementation
func (c *CliHumanEntity) ID() string {
	return c.id
}

func (c *CliHumanEntity) Name() string {
	return c.name
}

func (c *CliHumanEntity) Type() EntityType {
	return EntityTypeHuman
}

func (c *CliHumanEntity) Status() EntityStatus {
	return c.status
}

func (c *CliHumanEntity) SetStatus(status EntityStatus) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.status = status
	c.updatedAt = time.Now()
	return nil
}

func (c *CliHumanEntity) Metadata() Metadata {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.metadata
}

func (c *CliHumanEntity) SetMetadata(key string, value interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.metadata[key] = value
	c.updatedAt = time.Now()
	return nil
}

func (c *CliHumanEntity) Roles() []Role {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	roles := make([]Role, 0, len(c.roles))
	for role := range c.roles {
		roles = append(roles, role)
	}
	return roles
}

func (c *CliHumanEntity) HasRole(role Role) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, has := c.roles[role]
	return has
}

func (c *CliHumanEntity) AddRole(role Role) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.roles[role] = true
	c.updatedAt = time.Now()
	return nil
}

func (c *CliHumanEntity) RemoveRole(role Role) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.roles, role)
	c.updatedAt = time.Now()
	return nil
}

func (c *CliHumanEntity) CreatedAt() time.Time {
	return c.createdAt
}

func (c *CliHumanEntity) UpdatedAt() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.updatedAt
}

// processMessageAsResponse checks if this message is a response to an ongoing conversation
func (c *CliHumanEntity) processMessageAsResponse(msg messaging.Message) {
	// Check for any message metadata that identifies which message this might be responding to
	originalID := ""

	// Look in metadata for originalID or replyToID
	if val, exists := msg.Metadata["original_id"]; exists && val != "" {
		originalID = val
		c.logger.Debug("Found originalID in message metadata", "original_id", originalID, "response_id", msg.ID)
	} else if val, exists := msg.Metadata["reply_to_id"]; exists && val != "" {
		originalID = val
		c.logger.Debug("Found replyToID in message metadata", "reply_to_id", originalID, "response_id", msg.ID)
	}

	// If originalID was found, try to match with a pending conversation
	if originalID != "" {
		c.conversationMutex.RLock()
		_, isPending := c.pendingConversations[originalID]
		c.conversationMutex.RUnlock()

		if isPending {
			c.logger.Debug("Message is response to pending conversation", "original_id", originalID, "response_id", msg.ID)

			// Execute handler for the original message
			c.mutex.RLock()
			handler, exists := c.handlers[originalID]
			c.mutex.RUnlock()

			if exists {
				c.logger.Debug("Executing handler for original message", "original_id", originalID)
				handler(msg)

				// Remove the handler after execution
				c.mutex.Lock()
				delete(c.handlers, originalID)
				c.mutex.Unlock()

				// Remove from pending conversations
				c.conversationMutex.Lock()
				delete(c.pendingConversations, originalID)
				c.conversationMutex.Unlock()

				c.logger.Debug("Completed response handling", "original_id", originalID, "response_id", msg.ID)
			} else {
				c.logger.Debug("No handler for original message", "original_id", originalID)
			}
		}
	}
}

// GetPendingConversations returns a list of message IDs we're waiting for responses to
func (c *CliHumanEntity) GetPendingConversations() []string {
	c.conversationMutex.RLock()
	defer c.conversationMutex.RUnlock()

	result := make([]string, 0, len(c.pendingConversations))
	for id := range c.pendingConversations {
		result = append(result, id)
	}

	return result
}

func (c *CliHumanEntity) CanReceiveMessage() bool {
	return true
}

func (c *CliHumanEntity) CanSendMessage() bool {
	return true
}

func (c *CliHumanEntity) ReceiveMessage(msg messaging.Message) error {
	// This is handled by the message bus subscription
	return nil
}

func (c *CliHumanEntity) SendMessage(recipients []string, contentType string, content []byte) (messaging.Message, error) {
	// Create and send the message
	msg := messaging.NewMessage(c.id, recipients, contentType, content)
	c.logger.Debug("Human sending message",
		"entity_id", c.id,
		"message_id", msg.ID,
		"recipients", recipients,
		"content_length", len(content))

	// Publish to the message bus
	err := c.messageBus.Publish(msg)
	if err != nil {
		c.logger.Error("Failed to publish message", "message_id", msg.ID, "error", err)
	} else {
		c.logger.Debug("Message successfully published to bus", "message_id", msg.ID)
	}
	return msg, err
}
