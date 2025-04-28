package messaging

import (
	"fmt"
	"goproduct/internal/logging"
	"goproduct/internal/tracing"
	"sync"
	"time"
)

// MemoryMessageBus implements MessageBus using in-knowledge structures
type MemoryMessageBus struct {
	subscriptions map[string]MessageHandler
	groups        map[string]*Group
	tracer        tracing.Tracer
	logger        *logging.Logger
	mu            sync.RWMutex
}

// Group represents a message group with members
type Group struct {
	ID      string
	Name    string
	Members map[string]bool
}

// NewMemoryMessageBus creates a new in-knowledge message bus
func NewMemoryMessageBus() *MemoryMessageBus {
	return &MemoryMessageBus{
		subscriptions: make(map[string]MessageHandler),
		groups:        make(map[string]*Group),
		tracer:        tracing.NewNoopTracer(), // Default to no-op tracer
		logger:        logging.Get(),           // Use default logger
	}
}

// NewMemoryMessageBusWithTracer creates a new in-knowledge message bus with a custom tracer
func NewMemoryMessageBusWithTracer(tracer tracing.Tracer) *MemoryMessageBus {
	return &MemoryMessageBus{
		subscriptions: make(map[string]MessageHandler),
		groups:        make(map[string]*Group),
		tracer:        tracer,
		logger:        logging.Get(), // Use default logger
	}
}

// Publish sends a message to all its recipients
func (m *MemoryMessageBus) Publish(msg Message) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Track recipients for delivery
	delivered := make(map[string]bool)

	// Log the message being sent
	m.logger.Debug("Message published",
		"message_id", msg.ID,
		"sender", msg.SenderID,
		"content_type", msg.ContentType,
		"recipient_count", len(msg.Recipients))

	// Trace the message being sent (keeping existing tracing)
	m.tracer.Trace(tracing.Event{
		Timestamp: msg.Timestamp,
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationSend,
		Level:     tracing.LevelInfo,
		SourceID:  msg.SenderID,
		ObjectID:  msg.ID,
		Message:   "Message published",
		Metadata: map[string]interface{}{
			"contentType": msg.ContentType,
			"recipients":  msg.Recipients,
		},
	})

	// Handle each recipient
	for _, recipientID := range msg.Recipients {
		// Handle broadcast
		if recipientID == BroadcastAddress {
			for subID, handler := range m.subscriptions {
				if subID != msg.SenderID { // Don't send to self
					// Pass message to handler in a goroutine
					go func(recID string, h MessageHandler, message Message) {
						// Recover from panics in message handlers
						defer func() {
							if r := recover(); r != nil {
								m.tracer.Trace(tracing.Event{
									Timestamp: time.Now(),
									Component: tracing.ComponentMessaging,
									Operation: tracing.OperationReceive,
									Level:     tracing.LevelError,
									SourceID:  message.SenderID,
									TargetID:  recID,
									ObjectID:  message.ID,
									Message:   fmt.Sprintf("Panic in message handler: %v", r),
								})
								m.logger.Error("Panic in message handler",
									"error", r,
									"message_id", message.ID,
									"sender", message.SenderID,
									"recipient", recID)
							}
						}()

						// Log the message being received
						m.logger.Debug("Message received via broadcast",
							"message_id", message.ID,
							"sender", message.SenderID,
							"recipient", recID)

						// Trace the message being received
						m.tracer.Trace(tracing.Event{
							Timestamp: message.Timestamp,
							Component: tracing.ComponentMessaging,
							Operation: tracing.OperationReceive,
							Level:     tracing.LevelInfo,
							SourceID:  message.SenderID,
							TargetID:  recID,
							ObjectID:  message.ID,
							Message:   "Message received via broadcast",
						})

						// Call the handler and capture any error
						if err := h(message); err != nil {
							// Log the error
							m.logger.Error("Error in message handler",
								"error", err,
								"message_id", message.ID,
								"sender", message.SenderID,
								"recipient", recID)

							m.tracer.Trace(tracing.Event{
								Timestamp: time.Now(),
								Component: tracing.ComponentMessaging,
								Operation: tracing.OperationReceive,
								Level:     tracing.LevelError,
								SourceID:  message.SenderID,
								TargetID:  recID,
								ObjectID:  message.ID,
								Message:   fmt.Sprintf("Error in message handler: %v", err),
							})
						}
					}(subID, handler, msg)
					delivered[subID] = true
				}
			}
			continue
		}

		// Check if recipient is a group
		if group, ok := m.groups[recipientID]; ok {
			// Trace group message
			m.tracer.Trace(tracing.Event{
				Timestamp: msg.Timestamp,
				Component: tracing.ComponentMessaging,
				Operation: tracing.OperationSend,
				Level:     tracing.LevelInfo,
				SourceID:  msg.SenderID,
				TargetID:  recipientID, // Group ID
				ObjectID:  msg.ID,
				Message:   "Message sent to group",
			})

			// Deliver to each group member
			for memberID := range group.Members {
				if memberID != msg.SenderID { // Don't send to self
					if handler, exists := m.subscriptions[memberID]; exists {
						// Pass message to handler in a goroutine
						go func(recID string, h MessageHandler, message Message, grpID string) {
							// Recover from panics in message handlers
							defer func() {
								if r := recover(); r != nil {
									m.tracer.Trace(tracing.Event{
										Timestamp: time.Now(),
										Component: tracing.ComponentMessaging,
										Operation: tracing.OperationReceive,
										Level:     tracing.LevelError,
										SourceID:  message.SenderID,
										TargetID:  recID,
										ObjectID:  message.ID,
										Message:   fmt.Sprintf("Panic in group message handler: %v", r),
										Metadata: map[string]interface{}{
											"groupID": grpID,
										},
									})
									m.logger.Error("Panic in group message handler",
										"error", r,
										"message_id", message.ID,
										"sender", message.SenderID,
										"recipient", recID,
										"group_id", grpID)
								}
							}()

							// Log the message being received by a group member
							m.logger.Debug("Message received via group",
								"message_id", message.ID,
								"sender", message.SenderID,
								"recipient", recID,
								"group_id", grpID)

							// Trace the message being received by a group member
							m.tracer.Trace(tracing.Event{
								Timestamp: message.Timestamp,
								Component: tracing.ComponentMessaging,
								Operation: tracing.OperationReceive,
								Level:     tracing.LevelInfo,
								SourceID:  message.SenderID,
								TargetID:  recID,
								ObjectID:  message.ID,
								Message:   "Message received via group",
								Metadata: map[string]interface{}{
									"groupID": grpID,
								},
							})

							// Call the handler and capture any error
							if err := h(message); err != nil {
								// Log the error
								m.logger.Error("Error in group message handler",
									"error", err,
									"message_id", message.ID,
									"sender", message.SenderID,
									"recipient", recID,
									"group_id", grpID)

								m.tracer.Trace(tracing.Event{
									Timestamp: time.Now(),
									Component: tracing.ComponentMessaging,
									Operation: tracing.OperationReceive,
									Level:     tracing.LevelError,
									SourceID:  message.SenderID,
									TargetID:  recID,
									ObjectID:  message.ID,
									Message:   fmt.Sprintf("Error in group message handler: %v", err),
									Metadata: map[string]interface{}{
										"groupID": grpID,
									},
								})
							}
						}(memberID, handler, msg, recipientID)
						delivered[memberID] = true
					}
				}
			}
			continue
		}

		// Direct message to an entity
		if handler, ok := m.subscriptions[recipientID]; ok {
			// Pass message to handler in a goroutine
			go func(recID string, h MessageHandler, message Message) {
				// Recover from panics in message handlers
				defer func() {
					if r := recover(); r != nil {
						m.tracer.Trace(tracing.Event{
							Timestamp: time.Now(),
							Component: tracing.ComponentMessaging,
							Operation: tracing.OperationReceive,
							Level:     tracing.LevelError,
							SourceID:  message.SenderID,
							TargetID:  recID,
							ObjectID:  message.ID,
							Message:   fmt.Sprintf("Panic in direct message handler: %v", r),
						})
						m.logger.Error("Panic in direct message handler",
							"error", r,
							"message_id", message.ID,
							"sender", message.SenderID,
							"recipient", recID)
					}
				}()

				// Log the message being received
				m.logger.Debug("Message received directly",
					"message_id", message.ID,
					"sender", message.SenderID,
					"recipient", recID)

				// Trace the message being received
				m.tracer.Trace(tracing.Event{
					Timestamp: message.Timestamp,
					Component: tracing.ComponentMessaging,
					Operation: tracing.OperationReceive,
					Level:     tracing.LevelInfo,
					SourceID:  message.SenderID,
					TargetID:  recID,
					ObjectID:  message.ID,
					Message:   "Message received directly",
				})

				// Call the handler and capture any error
				if err := h(message); err != nil {
					// Log the error
					m.logger.Error("Error in direct message handler",
						"error", err,
						"message_id", message.ID,
						"sender", message.SenderID,
						"recipient", recID)

					m.tracer.Trace(tracing.Event{
						Timestamp: time.Now(),
						Component: tracing.ComponentMessaging,
						Operation: tracing.OperationReceive,
						Level:     tracing.LevelError,
						SourceID:  message.SenderID,
						TargetID:  recID,
						ObjectID:  message.ID,
						Message:   fmt.Sprintf("Error in direct message handler: %v", err),
					})
				}
			}(recipientID, handler, msg)
			delivered[recipientID] = true
		}
	}

	return nil
}

// Subscribe registers an entity to receive messages
func (m *MemoryMessageBus) Subscribe(entityID string, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate handler is not nil
	if handler == nil {
		return fmt.Errorf("message handler cannot be nil")
	}

	m.subscriptions[entityID] = handler

	// Log the subscription
	m.logger.Info("Entity subscribed to message bus", "entity_id", entityID)

	// Trace the subscription
	m.tracer.Trace(tracing.Event{
		Timestamp: time.Now(),
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationCreate,
		Level:     tracing.LevelInfo,
		TargetID:  entityID,
		Message:   "Entity subscribed to message bus",
	})

	return nil
}

// Unsubscribe removes an entity from receiving messages
func (m *MemoryMessageBus) Unsubscribe(entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscriptions, entityID)

	// Log the unsubscription
	m.logger.Info("Entity unsubscribed from message bus", "entity_id", entityID)

	// Trace the unsubscription
	m.tracer.Trace(tracing.Event{
		Timestamp: time.Now(),
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationDelete,
		Level:     tracing.LevelInfo,
		TargetID:  entityID,
		Message:   "Entity unsubscribed from message bus",
	})

	return nil
}

// CreateGroup creates a new message group
func (m *MemoryMessageBus) CreateGroup(groupID, name string, members []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.groups[groupID]; exists {
		return fmt.Errorf("group with ID %s already exists", groupID)
	}

	group := &Group{
		ID:      groupID,
		Name:    name,
		Members: make(map[string]bool),
	}

	for _, memberID := range members {
		group.Members[memberID] = true
	}

	m.groups[groupID] = group

	// Log group creation
	m.logger.Info("Message group created",
		"group_id", groupID,
		"name", name,
		"member_count", len(members))

	// Trace group creation
	m.tracer.Trace(tracing.Event{
		Timestamp: time.Now(),
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationCreate,
		Level:     tracing.LevelInfo,
		ObjectID:  groupID,
		Message:   "Message group created",
		Metadata: map[string]interface{}{
			"name":    name,
			"members": members,
		},
	})

	return nil
}

// AddToGroup adds an entity to a group
func (m *MemoryMessageBus) AddToGroup(groupID, entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupID]
	if !exists {
		return fmt.Errorf("group with ID %s does not exist", groupID)
	}

	group.Members[entityID] = true

	// Trace member addition
	m.tracer.Trace(tracing.Event{
		Timestamp: time.Now(),
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationJoin,
		Level:     tracing.LevelInfo,
		TargetID:  entityID,
		ObjectID:  groupID,
		Message:   "Entity added to message group",
	})

	return nil
}

// RemoveFromGroup removes an entity from a group
func (m *MemoryMessageBus) RemoveFromGroup(groupID, entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[groupID]
	if !exists {
		return fmt.Errorf("group with ID %s does not exist", groupID)
	}

	delete(group.Members, entityID)

	// Trace member removal
	m.tracer.Trace(tracing.Event{
		Timestamp: time.Now(),
		Component: tracing.ComponentMessaging,
		Operation: tracing.OperationLeave,
		Level:     tracing.LevelInfo,
		TargetID:  entityID,
		ObjectID:  groupID,
		Message:   "Entity removed from message group",
	})

	return nil
}

// GetGroupMembers returns all members of a group
func (m *MemoryMessageBus) GetGroupMembers(groupID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[groupID]
	if !exists {
		return nil, fmt.Errorf("group with ID %s does not exist", groupID)
	}

	members := make([]string, 0, len(group.Members))
	for memberID := range group.Members {
		members = append(members, memberID)
	}

	return members, nil
}

// SetTracer sets the tracer for this message bus
func (m *MemoryMessageBus) SetTracer(tracer tracing.Tracer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If nil tracer is provided, use a no-op tracer
	if tracer == nil {
		m.tracer = tracing.NewNoopTracer()
	} else {
		m.tracer = tracer
	}

	// Log tracer change
	m.logger.Debug("Message bus tracer updated")
}

// GetTracer returns the current tracer
func (m *MemoryMessageBus) GetTracer() tracing.Tracer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tracer
}

// SetLogger sets a custom logger for this message bus
func (m *MemoryMessageBus) SetLogger(logger *logging.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If nil logger is provided, use the default logger
	if logger == nil {
		m.logger = logging.Get()
	} else {
		m.logger = logger
	}
}

// GetLogger returns the current logger
func (m *MemoryMessageBus) GetLogger() *logging.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.logger
}
