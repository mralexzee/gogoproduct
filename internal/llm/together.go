package llm

import (
	"context"
	"errors"
	"fmt"
)

// TogetherLLM is the Together AI implementation of the LLM interface
type TogetherLLM struct {
	// Will use standard http client when implementing
	apiKey      string
	model       string // e.g., "togethercomputer/llama-3-8b"
	temperature float32
	maxTokens   int
}

// TogetherOption is a function that configures a TogetherLLM
type TogetherOption func(*TogetherLLM)

// NewTogetherLLM creates a new Together AI LLM with the specified options
func NewTogetherLLM(apiKey string, options ...TogetherOption) (*TogetherLLM, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Default values
	llm := &TogetherLLM{
		apiKey:      apiKey,
		model:       "togethercomputer/llama-3-8b",
		temperature: 0.7,
		maxTokens:   1024,
	}

	// Apply options
	for _, option := range options {
		option(llm)
	}

	return llm, nil
}

// WithTogetherModel sets the model for the Together AI LLM
func WithTogetherModel(model string) TogetherOption {
	return func(t *TogetherLLM) {
		t.model = model
	}
}

// WithTogetherTemperature sets the temperature for the Together AI LLM
func WithTogetherTemperature(temp float32) TogetherOption {
	return func(t *TogetherLLM) {
		t.temperature = temp
	}
}

// WithTogetherMaxTokens sets the max tokens for the Together AI LLM
func WithTogetherMaxTokens(maxTokens int) TogetherOption {
	return func(t *TogetherLLM) {
		t.maxTokens = maxTokens
	}
}

// GenerateResponse generates a text response for a single prompt
func (t *TogetherLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// This is a placeholder implementation
	// When we implement the real Together AI API calls, this would be replaced

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	return fmt.Sprintf("[Together AI %s] This is a placeholder response. The actual Together AI integration will be implemented later.", t.model), nil
}

// GenerateChat generates a response based on a conversation history
func (t *TogetherLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// This is a placeholder implementation
	// Together AI supports chat-style interactions with a specific format

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

	return fmt.Sprintf("[Together AI %s] Chat response placeholder for prompt: %s", t.model, lastMessage), nil
}

// Initialize the factory function
func init() {
	newTogetherFromConfig = createTogetherFromConfig
}

// Implementation for creating a TogetherLLM from config
func createTogetherFromConfig(config ProviderConfig) (LanguageModel, error) {
	// Check if it's specifically a TogetherConfig
	togetherConfig, ok := config.(*TogetherConfig)
	if !ok {
		// Try to extract basic information from the generic config
		baseConfig, ok := config.(BaseConfig)
		if !ok {
			return nil, errors.New("invalid configuration type for Together AI")
		}

		// Get API key from environment if not in config
		apiKey := ""
		if envAPIKey := getEnvWithDefault("TOGETHER_API_KEY", ""); envAPIKey != "" {
			apiKey = envAPIKey
		} else {
			return nil, ErrAPIKeyMissing
		}

		return NewTogetherLLM(
			apiKey,
			WithTogetherModel(baseConfig.Model),
			WithTogetherTemperature(baseConfig.Temperature),
			WithTogetherMaxTokens(baseConfig.MaxTokens),
		)
	}

	// Use the specific Together AI configuration
	options := []TogetherOption{
		WithTogetherModel(togetherConfig.Model),
		WithTogetherTemperature(togetherConfig.Temperature),
		WithTogetherMaxTokens(togetherConfig.MaxTokens),
	}

	return NewTogetherLLM(togetherConfig.APIKey, options...)
}
