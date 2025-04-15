package agent

import (
	"context"
	"goproduct/internal/llm"
)

// MockLLM is a mock implementation of the LanguageModel interface for testing
type MockLLM struct{}

// GenerateChat returns a simple echo response for testing
func (m *MockLLM) GenerateChat(ctx context.Context, messages []llm.Message) (string, error) {
	// For testing, just echo the last user message
	var lastMessage string
	for _, msg := range messages {
		if msg.Role == "user" {
			lastMessage = msg.Content
		}
	}
	return "ECHO: ECHO: " + lastMessage, nil
}

// GenerateResponse implements the LanguageModel interface
func (m *MockLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return "ECHO: ECHO: " + prompt, nil
}

// Name returns the name of the mock LLM
func (m *MockLLM) Name() string {
	return "MockLLM"
}

// Type returns the type of the mock LLM
func (m *MockLLM) Type() string {
	return "mock"
}
