package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMockLLM(t *testing.T) {
	// Create a mock LLM with default settings
	mockLLM := NewMockLLM()

	// Test GenerateResponse
	resp, err := mockLLM.GenerateResponse(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	expected := "Thank you for asking me: Hello, world!"
	if resp != expected {
		t.Errorf("Expected response: %q, got: %q", expected, resp)
	}

	// Test history recording
	history := mockLLM.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 interaction in history, got %d", len(history))
	}

	if history[0].Prompt != "Hello, world!" {
		t.Errorf("Expected prompt in history: %q, got: %q", "Hello, world!", history[0].Prompt)
	}

	// Test with custom prefix
	customMockLLM := NewMockLLM(WithResponsePrefix("I am a test assistant. You asked: "))

	resp, err = customMockLLM.GenerateResponse(context.Background(), "Test question")
	if err != nil {
		t.Fatalf("Failed to generate response with custom prefix: %v", err)
	}

	expected = "I am a test assistant. You asked: Test question"
	if resp != expected {
		t.Errorf("Expected custom response: %q, got: %q", expected, resp)
	}

	// Test GenerateChat
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is the capital of France?"},
	}

	resp, err = mockLLM.GenerateChat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Failed to generate chat response: %v", err)
	}

	expected = "Thank you for asking me: What is the capital of France?"
	if resp != expected {
		t.Errorf("Expected chat response: %q, got: %q", expected, resp)
	}

	// Test with delay
	delayMockLLM := NewMockLLM(WithDelay(100 * time.Millisecond))

	start := time.Now()
	resp, err = delayMockLLM.GenerateResponse(context.Background(), "Delayed question")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to generate delayed response: %v", err)
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected delay of at least 100ms, got %v", elapsed)
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = delayMockLLM.GenerateResponse(ctx, "Cancelled question")
	if err == nil {
		t.Errorf("Expected error due to cancelled context, got nil")
	}

	// Test fixed response
	fixedMockLLM := NewMockLLM(WithFixedResponse("This is a fixed response"))

	resp, err = fixedMockLLM.GenerateResponse(context.Background(), "Any question")
	if err != nil {
		t.Fatalf("Failed to generate fixed response: %v", err)
	}

	expected = "This is a fixed response"
	if resp != expected {
		t.Errorf("Expected fixed response: %q, got: %q", expected, resp)
	}

	// Test history clearing
	mockLLM.ClearHistory()
	history = mockLLM.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clearing, got %d entries", len(history))
	}

	// Test factory function
	mockConfig := LoadMockConfig()
	mockConfig.ResponsePrefix = "Factory test: "

	llm, err := newMockFromConfig(mockConfig)
	if err != nil {
		t.Fatalf("Failed to create mock from config: %v", err)
	}

	resp, err = llm.GenerateResponse(context.Background(), "Factory question")
	if err != nil {
		t.Fatalf("Failed to generate response from factory-created mock: %v", err)
	}

	if !strings.Contains(resp, "Factory test:") {
		t.Errorf("Expected factory response to contain custom prefix, got: %q", resp)
	}
}
