package llm

import (
	"context"
	"fmt"
	"strings"
)

// Provider constants
const (
	ProviderOpenAI      = "openai"
	ProviderAnthropicAI = "anthropic"
	ProviderOllama      = "ollama"
	ProviderTogetherAI  = "together"
	ProviderLMStudio    = "lmstudio"
	ProviderMock        = "mock"
)

// NewLLM creates a new LLM instance based on the provided configuration
func NewLLM(ctx context.Context, config ProviderConfig) (LanguageModel, error) {
	switch strings.ToLower(config.GetProvider()) {
	case ProviderOpenAI:
		return newOpenAIFromConfig(config)
	case ProviderAnthropicAI:
		return newAnthropicFromConfig(config)
	case ProviderOllama:
		return newOllamaFromConfig(config)
	case ProviderTogetherAI:
		return newTogetherFromConfig(config)
	case ProviderLMStudio:
		return newLMStudioFromConfig(config)
	case ProviderMock:
		return newMockFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.GetProvider())
	}
}

// Factory functions are declared here for documentation, but implemented in their respective files

// newOpenAIFromConfig creates an OpenAI LLM from configuration
// Implemented in openai.go
var newOpenAIFromConfig func(config ProviderConfig) (LanguageModel, error)

// newAnthropicFromConfig creates an Anthropic LLM from configuration
// Implemented in anthropic.go
var newAnthropicFromConfig func(config ProviderConfig) (LanguageModel, error)

// newOllamaFromConfig creates an Ollama LLM from configuration
// Implemented in ollama.go
var newOllamaFromConfig func(config ProviderConfig) (LanguageModel, error)

// newTogetherFromConfig creates a Together AI LLM from configuration
// Implemented in together.go
var newTogetherFromConfig func(config ProviderConfig) (LanguageModel, error)

// newLMStudioFromConfig creates an LM Studio LLM from configuration
// Implemented in lmstudio.go
var newLMStudioFromConfig func(config ProviderConfig) (LanguageModel, error)

// newMockFromConfig creates a Mock LLM from configuration
// Implemented in mock.go
var newMockFromConfig func(config ProviderConfig) (LanguageModel, error)
