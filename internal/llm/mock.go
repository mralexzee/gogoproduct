package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockLLM implements LLM interface for testing purposes
type MockLLM struct {
	responsePrefix string            // Customizable prefix for responses
	delay          time.Duration     // Optional delay to simulate processing time
	responseFormat string            // Format string for responses
	history        []MockInteraction // Record of interactions for verification
}

// MockInteraction records an interaction for test verification
type MockInteraction struct {
	Prompt   string
	Messages []Message
	Response string
	Time     time.Time
}

// NewMockLLM creates a new mock LLM with the specified options
func NewMockLLM(options ...MockOption) *MockLLM {
	// Default values
	mock := &MockLLM{
		responsePrefix: "Thank you for asking me: ",
		delay:          0,      // No delay by default
		responseFormat: "%s%s", // Default format is prefix + prompt
		history:        make([]MockInteraction, 0),
	}

	// Apply options
	for _, option := range options {
		option(mock)
	}

	return mock
}

// GenerateResponse implements the LLM interface for a single prompt
func (m *MockLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Optional delay to simulate processing
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Format response using template
	var response string
	if m.responseFormat == "%s" {
		// Fixed response case - only use the prefix
		response = m.responsePrefix
	} else {
		// Normal case - format with prefix and prompt
		response = fmt.Sprintf(m.responseFormat, m.responsePrefix, prompt)
	}

	// Record interaction for test verification
	m.history = append(m.history, MockInteraction{
		Prompt:   prompt,
		Response: response,
		Time:     time.Now(),
	})

	return response, nil
}

// GenerateChat implements the LLM interface for a conversation
func (m *MockLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// Extract the last user message as the prompt
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	// If no user message found, use a summary of the conversation
	if lastUserMessage == "" {
		var sb strings.Builder
		for _, msg := range messages {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		}
		lastUserMessage = fmt.Sprintf("Conversation summary: %s", sb.String())
	}

	// Generate response using the same logic
	response, err := m.GenerateResponse(ctx, lastUserMessage)

	// Record the full conversation
	if err == nil {
		m.history = append(m.history, MockInteraction{
			Messages: messages,
			Response: response,
			Time:     time.Now(),
		})
	}

	return response, err
}

// GetHistory returns the history of interactions
func (m *MockLLM) GetHistory() []MockInteraction {
	return m.history
}

// ClearHistory clears the interaction history
func (m *MockLLM) ClearHistory() {
	m.history = make([]MockInteraction, 0)
}

// MockOption is a function that configures a MockLLM
type MockOption func(*MockLLM)

// WithResponsePrefix sets the response prefix
func WithResponsePrefix(prefix string) MockOption {
	return func(m *MockLLM) {
		m.responsePrefix = prefix
	}
}

// WithDelay sets the delay before responding
func WithDelay(delay time.Duration) MockOption {
	return func(m *MockLLM) {
		m.delay = delay
	}
}

// WithResponseFormat sets the response format
func WithResponseFormat(format string) MockOption {
	return func(m *MockLLM) {
		m.responseFormat = format
	}
}

// WithFixedResponse sets a fixed response regardless of input
func WithFixedResponse(response string) MockOption {
	return func(m *MockLLM) {
		// For fixed responses, use a format that doesn't interpolate the prompt
		m.responsePrefix = response
		m.responseFormat = "%s" // Only use the prefix, ignore the prompt
	}
}

// Initialize the factory function
func init() {
	newMockFromConfig = createMockFromConfig
}

// Implementation for creating a MockLLM from config
func createMockFromConfig(config ProviderConfig) (LanguageModel, error) {
	var options []MockOption

	// If it's specifically a MockConfig, use those options
	if mockConfig, ok := config.(*MockConfig); ok {
		options = append(options, WithResponsePrefix(mockConfig.ResponsePrefix))
		if mockConfig.DelayMs > 0 {
			options = append(options, WithDelay(time.Duration(mockConfig.DelayMs)*time.Millisecond))
		}
	}

	return NewMockLLM(options...), nil
}
