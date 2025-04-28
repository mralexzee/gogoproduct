package agent

import (
	"context"
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	persona := Persona{
		Name: "TestAgent",
		Role: "Assistant",
		Type: "Test",
	}

	agent := NewAgent(persona)

	if agent == nil {
		t.Fatal("Expected agent to be created, got nil")
	}

	if agent.Persona.Name != persona.Name {
		t.Errorf("Expected agent name to be %s, got %s", persona.Name, agent.Persona.Name)
	}

	if agent.Persona.Role != persona.Role {
		t.Errorf("Expected agent role to be %s, got %s", persona.Role, agent.Persona.Role)
	}

	if agent.Persona.Type != persona.Type {
		t.Errorf("Expected agent type to be %s, got %s", persona.Type, agent.Persona.Type)
	}

	if agent._messages == nil {
		t.Error("Expected _messages channel to be initialized")
	}

	if agent.stopCh == nil {
		t.Error("Expected stopCh to be initialized")
	}
}

func TestAgentChat(t *testing.T) {
	// Create a test agent with mock LLM
	persona := Persona{
		Name: "TestAgent",
		Role: "Assistant",
		Type: "Test",
		LanguageModels: LanguageModels{
			Default: &MockLLM{},
		},
	}
	agent := NewAgent(persona)

	// Create context with shorter timeout for tests
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start the agent
	agent.Start(ctx)

	// Send a chat message
	message := "Hello, Test Agent!"
	sender := "TestUser"

	msg := agent.Chat(sender, message)

	// Verify the message fields
	if msg.Content != message {
		t.Errorf("Expected message content to be '%s', got '%s'", message, msg.Content)
	}

	if msg.From != sender {
		t.Errorf("Expected sender to be '%s', got '%s'", sender, msg.From)
	}

	if len(msg.To) != 1 || msg.To[0] != persona.Name {
		t.Errorf("Expected recipient to be '%s', got '%v'", persona.Name, msg.To)
	}

	if msg.Type != "chat" {
		t.Errorf("Expected message type to be 'chat', got '%s'", msg.Type)
	}

	// Wait for response (end-to-end test)
	select {
	case response := <-msg.ResponseReady:
		// The actual content is formatted by the agent which adds another "ECHO: " prefix
		expectedPrefix := "ECHO: ECHO: " + message
		if response.Content != expectedPrefix {
			t.Errorf("Expected response content to be '%s', got '%s'", expectedPrefix, response.Content)
		}
	case <-time.After(1 * time.Second): // Reduced timeout for tests
		t.Fatal("Timed out waiting for response")
	}

	// Clean up
	agent.Stop()
}

func TestAgentStop(t *testing.T) {
	persona := Persona{
		Name: "TestAgent",
		Role: "Assistant",
		Type: "Test",
		LanguageModels: LanguageModels{
			Default: &MockLLM{},
		},
	}
	agent := NewAgent(persona)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the agent
	agent.Start(ctx)

	// Stop the agent
	agent.Stop()

	// Verify we can restart without errors (tests proper cleanup)
	agent.Start(ctx)
	agent.Stop()
}

func TestContextCancellation(t *testing.T) {
	persona := Persona{
		Name: "TestAgent",
		Role: "Assistant",
		Type: "Test",
		LanguageModels: LanguageModels{
			Default: &MockLLM{},
		},
	}
	agent := NewAgent(persona)

	ctx, cancel := context.WithCancel(context.Background())

	// Start the agent
	agent.Start(ctx)

	// Cancel the context (should stop the worker)
	cancel()

	// Give a small amount of time for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Verify we can restart without errors after context cancellation
	ctx2 := context.Background()
	agent.Start(ctx2)
	agent.Stop()
}
