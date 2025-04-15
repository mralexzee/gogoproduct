package agent

import (
	"context"
	"fmt"
	"goproduct/internal/llm"
	"goproduct/internal/logging"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	Persona   Persona
	stopCh    chan struct{}
	_messages chan Message
	_history  []llm.Message
	logger    *logging.Logger
}

func (a *Agent) Start(ctx context.Context) {
	// Ensure logger is initialized
	if a.logger == nil {
		a.logger = logging.Get()
	}

	a.logger.Info("Agent starting", "name", a.Persona.Name, "role", a.Persona.Role)
	a.stopCh = make(chan struct{})
	go a.worker(ctx)
}

func (a *Agent) Stop() {
	if a.logger != nil {
		a.logger.Info("Agent stopping", "name", a.Persona.Name)
	}
	close(a.stopCh)
}

func (a *Agent) worker(ctx context.Context) {
	a.logger.Debug("Agent worker started", "name", a.Persona.Name)
	for {
		select {
		case <-ctx.Done():
			a.logger.Debug("Agent worker stopped due to context done", "name", a.Persona.Name)
			return
		case <-a.stopCh:
			a.logger.Debug("Agent worker stopped due to stop channel", "name", a.Persona.Name)
			return
		case msg := <-a._messages:
			a.logger.Debug("Agent received message",
				"message_id", msg.Id,
				"message_type", msg.Type,
				"from", msg.From)
			a.handleMessage(msg)
		}
	}
}

func (a *Agent) handleMessage(msg Message) {
	switch msg.Type {
	case "chat":
		a.logger.Debug("Handling chat message", "message_id", msg.Id)
		a.handleChat(msg)
	default:
		a.logger.Warn("Received unknown message type", "message_id", msg.Id, "type", msg.Type)
	}
}

func (a *Agent) handleChat(msg Message) {
	a.logger.Debug("Processing chat message",
		"message_id", msg.Id,
		"from", msg.From,
		"content_length", len(msg.Content))

	if len(a._history) == 0 {
		a.logger.Debug("Initializing chat history with system prompt",
			"prompt_length", len(a.Persona.SystemPrompt))
		a._history = append(a._history, llm.Message{
			Role:    "system",
			Content: a.Persona.SystemPrompt,
		})
	}

	a._history = append(a._history, llm.Message{
		Role:    "user",
		Content: msg.Content,
	})

	a.logger.Debug("Generating LLM response",
		"message_id", msg.Id,
		"history_length", len(a._history))

	response, err := a.Persona.LanguageModels.Default.GenerateChat(context.Background(), a._history)
	if err != nil {
		a.handleLLMError(msg, err)
		return
	}

	a.logger.Debug("LLM response received",
		"message_id", msg.Id,
		"response_length", len(response))

	a._history = append(a._history, llm.Message{
		Role:    "assistant",
		Content: response,
	})

	// Create a proper response message with a new ID that references the original
	responseContent := fmt.Sprintf("%s", response)
	responseMsg := Message{
		Content:       responseContent,
		From:          a.Persona.Name,
		To:            []string{msg.From},
		Type:          "chat",
		ResponseReady: msg.ResponseReady,
		Created:       time.Now(),
		Id:            uuid.New().String(), // Always use a new ID
		OriginalId:    msg.Id,              // Reference original message
	}

	a.logger.Debug("Sending response to requester",
		"message_id", responseMsg.Id,
		"references_message", msg.Id,
		"to", msg.From)

	// Send the response through the channel
	msg.ResponseReady <- responseMsg
}

func (a *Agent) handleLLMError(msg Message, err error) {
	a.logger.Error("LLM generation failed",
		"error", err,
		"message_id", msg.Id)

	// Prepare fallback response text
	content := "I'm out of office today. If you need immediate assistance, please contact Tom Reynolds."

	// Create a proper response message that references the original
	responseID := uuid.New().String()
	responseMsg := Message{
		Content:       content,
		From:          a.Persona.Name,
		To:            []string{msg.From},
		Type:          "chat",
		ResponseReady: msg.ResponseReady,
		Created:       time.Now(),
		Id:            responseID, // Always use a new ID
		OriginalId:    msg.Id,     // Reference original message
	}

	a.logger.Debug("Sending fallback response",
		"message_id", responseMsg.Id,
		"recipient", msg.From,
		"references_message", msg.Id)

	// Log details to help debug
	a.logger.Debug("Response message details",
		"message_id", responseMsg.Id,
		"from", a.Persona.Name,
		"to", msg.From,
		"response_channel_exists", msg.ResponseReady != nil,
		"original_message_id", msg.Id)

	// Send the response message through the original response channel
	msg.ResponseReady <- responseMsg
}

func (a *Agent) Chat(from string, message string) Message {
	id := uuid.New().String() // Generate a unique ID

	a.logger.Debug("Creating new chat message",
		"message_id", id,
		"from", from,
		"to", a.Persona.Name,
		"content_length", len(message))

	msg := Message{
		Content:       message,
		From:          from,
		To:            []string{a.Persona.Name},
		Type:          "chat",
		ResponseReady: make(chan Message, 1), // Buffered channel
		Created:       time.Now(),
		Id:            id,
	}

	a.logger.Debug("Sending message to agent processing queue", "message_id", id)
	a._messages <- msg
	return msg
}

// HandleExternalMessage allows processing a message created externally
// Used by entity adapters to bridge the messaging system
func (a *Agent) HandleExternalMessage(msg Message) {
	// Ensure logger is initialized
	if a.logger == nil {
		a.logger = logging.Get()
	}

	a.logger.Debug("Handling external message",
		"message_id", msg.Id,
		"from", msg.From,
		"type", msg.Type)

	// Process the message through normal flow
	a.logger.Debug("Forwarding external message to internal queue", "message_id", msg.Id)
	a._messages <- msg
}

func NewAgent(p Persona) *Agent {
	logger := logging.Get()
	logger.Info("Creating new agent", "name", p.Name, "role", p.Role)

	return &Agent{
		Persona:   p,
		stopCh:    make(chan struct{}),
		_messages: make(chan Message, 100),
		_history:  make([]llm.Message, 0, 100),
		logger:    logger,
	}
}

type Message struct {
	Content       string       `json:"content"`
	Created       time.Time    `json:"created"`
	Id            string       `json:"id"`
	From          string       `json:"from"`
	To            []string     `json:"to"`
	Type          string       `json:"type"`
	ResponseReady chan Message `json:"response_ready"`
	OriginalId    string       `json:"original_id,omitempty"` // References original message in a conversation
}

type Persona struct {
	Name           string         `json:"name"`
	Role           string         `json:"role"`
	Type           string         `json:"type"`
	SystemPrompt   string         `json:"system_prompt"`
	LanguageModels LanguageModels `json:"language_models"`
}

type LanguageModels struct {
	Default llm.LanguageModel
}
