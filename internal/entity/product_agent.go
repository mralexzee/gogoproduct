package entity

import (
	"context"
	"goproduct/internal/agent"
	"goproduct/internal/messaging"
	"time"

	"github.com/google/uuid"
)

// ProductAgentEntity is an adapter that wraps the existing agent implementation
// to implement the Entity interface
type ProductAgentEntity struct {
	id         string
	name       string
	status     EntityStatus
	createdAt  time.Time
	updatedAt  time.Time
	agent      *agent.Agent
	messageBus messaging.MessageBus
	roles      map[Role]bool
	metadata   Metadata
}

// NewProductAgentEntity creates a new product agent entity
func NewProductAgentEntity(agent *agent.Agent, bus messaging.MessageBus) *ProductAgentEntity {
	now := time.Now()
	return &ProductAgentEntity{
		id:         uuid.New().String(),
		name:       agent.Persona.Name,
		status:     StatusActive,
		createdAt:  now,
		updatedAt:  now,
		agent:      agent,
		messageBus: bus,
		roles:      map[Role]bool{RoleDeveloper: true},
		metadata:   make(Metadata),
	}
}

// Start initializes the product agent and subscribes to messages
func (p *ProductAgentEntity) Start(ctx context.Context) error {
	// Start the underlying agent
	p.agent.Start(ctx)

	// Subscribe to messages
	return p.messageBus.Subscribe(p.id, func(msg messaging.Message) error {
		// Convert to agent message
		agentMsg := agent.Message{
			Id:            msg.ID,
			Content:       string(msg.Content),
			Created:       msg.Timestamp,
			From:          msg.SenderID,
			To:            []string{p.name},
			Type:          "chat",
			ResponseReady: make(chan agent.Message, 1),
		}

		// Process the message using the underlying agent
		go func() {
			p.agent.HandleExternalMessage(agentMsg)

			// Wait for response with a timeout
			select {
			case response := <-agentMsg.ResponseReady:
				// Send response back through message bus
				responseMsg := messaging.NewTextMessage(
					p.id,
					[]string{msg.SenderID},
					response.Content,
				)

				// Set as reply to original message
				responseMsg = responseMsg.WithReplyTo(msg.ID)

				// Set the original_id metadata field if agent set OriginalId
				if response.OriginalId != "" {
					responseMsg.Metadata["original_id"] = response.OriginalId
				}

				// Send response
				p.messageBus.Publish(responseMsg)

			case <-time.After(30 * time.Second):
				// If no response after timeout, send a fallback message
				responseMsg := messaging.NewTextMessage(
					p.id,
					[]string{msg.SenderID},
					"I seem to be having technical difficulties. Please try again later or contact Tom Reynolds.",
				)
				responseMsg = responseMsg.WithReplyTo(msg.ID)
				p.messageBus.Publish(responseMsg)
			}
		}()

		return nil
	})
}

// Shutdown stops the product agent
func (p *ProductAgentEntity) Shutdown() error {
	p.agent.Stop()
	return p.messageBus.Unsubscribe(p.id)
}

// Entity interface implementation
func (p *ProductAgentEntity) ID() string {
	return p.id
}

func (p *ProductAgentEntity) Name() string {
	return p.name
}

func (p *ProductAgentEntity) Type() EntityType {
	return EntityTypeAgent
}

func (p *ProductAgentEntity) Status() EntityStatus {
	return p.status
}

func (p *ProductAgentEntity) SetStatus(status EntityStatus) error {
	p.status = status
	p.updatedAt = time.Now()
	return nil
}

func (p *ProductAgentEntity) Metadata() Metadata {
	return p.metadata
}

func (p *ProductAgentEntity) SetMetadata(key string, value interface{}) error {
	p.metadata[key] = value
	p.updatedAt = time.Now()
	return nil
}

func (p *ProductAgentEntity) Roles() []Role {
	roles := make([]Role, 0, len(p.roles))
	for role := range p.roles {
		roles = append(roles, role)
	}
	return roles
}

func (p *ProductAgentEntity) HasRole(role Role) bool {
	_, has := p.roles[role]
	return has
}

func (p *ProductAgentEntity) AddRole(role Role) error {
	p.roles[role] = true
	p.updatedAt = time.Now()
	return nil
}

func (p *ProductAgentEntity) RemoveRole(role Role) error {
	delete(p.roles, role)
	p.updatedAt = time.Now()
	return nil
}

func (p *ProductAgentEntity) CreatedAt() time.Time {
	return p.createdAt
}

func (p *ProductAgentEntity) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *ProductAgentEntity) CanReceiveMessage() bool {
	return true
}

func (p *ProductAgentEntity) CanSendMessage() bool {
	return true
}

func (p *ProductAgentEntity) ReceiveMessage(msg messaging.Message) error {
	// This is handled by the message bus subscription
	return nil
}

func (p *ProductAgentEntity) SendMessage(recipients []string, contentType string, content []byte) (messaging.Message, error) {
	// Create message
	msg := messaging.NewMessage(p.id, recipients, contentType, content)

	// Send via message bus
	err := p.messageBus.Publish(msg)
	return msg, err
}
