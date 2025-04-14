package llm

import (
	"context"
	"errors"
	"fmt"
)

// OpenAILLM is the OpenAI implementation of the LLM interface
type OpenAILLM struct {
	// To be replaced with actual OpenAI client when dependency is added
	// client      *openai.Client
	apiKey       string
	organization string
	model        string
	temperature  float32
	maxTokens    int
}

// OpenAIOption is a function that configures an OpenAILLM
type OpenAIOption func(*OpenAILLM)

// NewOpenAILLM creates a new OpenAI LLM with the specified options
func NewOpenAILLM(apiKey string, options ...OpenAIOption) (*OpenAILLM, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Default values
	llm := &OpenAILLM{
		apiKey:      apiKey,
		model:       "gpt-4o",
		temperature: 0.7,
		maxTokens:   1024,
	}

	// Apply options
	for _, option := range options {
		option(llm)
	}

	// Initialize client
	// llm.client = openai.NewClient(apiKey)
	// if llm.organization != "" {
	//    llm.client.Organization = llm.organization
	// }

	return llm, nil
}

// WithOpenAIModel sets the model for the OpenAI LLM
func WithOpenAIModel(model string) OpenAIOption {
	return func(o *OpenAILLM) {
		o.model = model
	}
}

// WithOpenAITemperature sets the temperature for the OpenAI LLM
func WithOpenAITemperature(temp float32) OpenAIOption {
	return func(o *OpenAILLM) {
		o.temperature = temp
	}
}

// WithOpenAIMaxTokens sets the max tokens for the OpenAI LLM
func WithOpenAIMaxTokens(maxTokens int) OpenAIOption {
	return func(o *OpenAILLM) {
		o.maxTokens = maxTokens
	}
}

// WithOpenAIOrganization sets the organization for the OpenAI LLM
func WithOpenAIOrganization(org string) OpenAIOption {
	return func(o *OpenAILLM) {
		o.organization = org
	}
}

// GenerateResponse generates a text response for a single prompt
func (o *OpenAILLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// This is a placeholder implementation
	// When the OpenAI Go SDK is added as a dependency, this will be replaced with actual API calls

	// Convert prompt to messages for the chat API
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	return o.GenerateChat(ctx, messages)
}

// GenerateChat generates a response based on a conversation history
func (o *OpenAILLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// This is a placeholder implementation
	// When the OpenAI Go SDK is added as a dependency, this will be replaced with actual API calls

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	// For now, we'll return a message indicating this is a placeholder
	return fmt.Sprintf("[OpenAI %s] This is a placeholder response. The actual OpenAI integration requires adding the OpenAI Go SDK as a dependency.", o.model), nil
}

// Initialize the factory function
func init() {
	newOpenAIFromConfig = createOpenAIFromConfig
}

// Implementation for creating an OpenAILLM from config
func createOpenAIFromConfig(config ProviderConfig) (LanguageModel, error) {
	// Check if it's specifically an OpenAIConfig
	openAIConfig, ok := config.(*OpenAIConfig)
	if !ok {
		// Try to extract basic information from the generic config
		baseConfig, ok := config.(BaseConfig)
		if !ok {
			return nil, errors.New("invalid configuration type for OpenAI")
		}

		// Get API key from environment if not in config
		apiKey := ""
		if envAPIKey := getEnvWithDefault("OPENAI_API_KEY", ""); envAPIKey != "" {
			apiKey = envAPIKey
		} else {
			return nil, ErrAPIKeyMissing
		}

		return NewOpenAILLM(
			apiKey,
			WithOpenAIModel(baseConfig.Model),
			WithOpenAITemperature(baseConfig.Temperature),
			WithOpenAIMaxTokens(baseConfig.MaxTokens),
		)
	}

	// Use the specific OpenAI configuration
	options := []OpenAIOption{
		WithOpenAIModel(openAIConfig.Model),
		WithOpenAITemperature(openAIConfig.Temperature),
		WithOpenAIMaxTokens(openAIConfig.MaxTokens),
	}

	if openAIConfig.Organization != "" {
		options = append(options, WithOpenAIOrganization(openAIConfig.Organization))
	}

	return NewOpenAILLM(openAIConfig.APIKey, options...)
}
