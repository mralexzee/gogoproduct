package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// OllamaLLM is the Ollama implementation of the LLM interface
type OllamaLLM struct {
	// Will use standard http client for now
	client      *http.Client
	endpoint    string // Usually "http://localhost:11434"
	model       string // e.g., "llama3", "mistral"
	temperature float32
	maxTokens   int
}

// OllamaOption is a function that configures an OllamaLLM
type OllamaOption func(*OllamaLLM)

// NewOllamaLLM creates a new Ollama LLM with the specified options
func NewOllamaLLM(endpoint string, options ...OllamaOption) (*OllamaLLM, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434" // Default endpoint
	}

	// Default values
	llm := &OllamaLLM{
		client:      &http.Client{},
		endpoint:    endpoint,
		model:       "llama3",
		temperature: 0.7,
		maxTokens:   1024,
	}

	// Apply options
	for _, option := range options {
		option(llm)
	}

	// Validate endpoint
	if !strings.HasPrefix(llm.endpoint, "http") {
		return nil, fmt.Errorf("invalid Ollama endpoint: %s, must start with http or https", llm.endpoint)
	}

	return llm, nil
}

// WithOllamaModel sets the model for the Ollama LLM
func WithOllamaModel(model string) OllamaOption {
	return func(o *OllamaLLM) {
		o.model = model
	}
}

// WithOllamaTemperature sets the temperature for the Ollama LLM
func WithOllamaTemperature(temp float32) OllamaOption {
	return func(o *OllamaLLM) {
		o.temperature = temp
	}
}

// WithOllamaMaxTokens sets the max tokens for the Ollama LLM
func WithOllamaMaxTokens(maxTokens int) OllamaOption {
	return func(o *OllamaLLM) {
		o.maxTokens = maxTokens
	}
}

// GenerateResponse generates a text response for a single prompt
func (o *OllamaLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// This is a placeholder implementation
	// When we implement the real Ollama API calls, this would be replaced

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	return fmt.Sprintf("[Ollama %s] This is a placeholder response. The actual Ollama integration will use the HTTP API at %s.", o.model, o.endpoint), nil
}

// GenerateChat generates a response based on a conversation history
func (o *OllamaLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// This is a placeholder implementation
	// Ollama supports chat-style interactions with a specific format

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	// Extract the last user message for demonstration
	var lastMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastMessage = messages[i].Content
			break
		}
	}

	if lastMessage == "" {
		lastMessage = "No user message found in conversation"
	}

	return fmt.Sprintf("[Ollama %s] Chat response placeholder for prompt: %s", o.model, lastMessage), nil
}

// Initialize the factory function
func init() {
	newOllamaFromConfig = createOllamaFromConfig
}

// Implementation for creating an OllamaLLM from config
func createOllamaFromConfig(config ProviderConfig) (LanguageModel, error) {
	// Check if it's specifically an OllamaConfig
	ollamaConfig, ok := config.(*OllamaConfig)
	if !ok {
		// Try to extract basic information from the generic config
		baseConfig, ok := config.(BaseConfig)
		if !ok {
			return nil, errors.New("invalid configuration type for Ollama")
		}

		// Get endpoint from environment if not in config
		endpoint := getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434")

		return NewOllamaLLM(
			endpoint,
			WithOllamaModel(baseConfig.Model),
			WithOllamaTemperature(baseConfig.Temperature),
			WithOllamaMaxTokens(baseConfig.MaxTokens),
		)
	}

	// Use the specific Ollama configuration
	options := []OllamaOption{
		WithOllamaModel(ollamaConfig.Model),
		WithOllamaTemperature(ollamaConfig.Temperature),
		WithOllamaMaxTokens(ollamaConfig.MaxTokens),
	}

	return NewOllamaLLM(ollamaConfig.Endpoint, options...)
}
