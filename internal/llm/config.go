package llm

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

// OpenAIConfig contains OpenAI-specific configuration
type OpenAIConfig struct {
	BaseConfig
	APIKey       string
	Organization string
}

// AnthropicConfig contains Anthropic-specific configuration
type AnthropicConfig struct {
	BaseConfig
	APIKey string
}

// OllamaConfig contains Ollama-specific configuration
type OllamaConfig struct {
	BaseConfig
	Endpoint string // Usually http://localhost:11434
}

// TogetherConfig contains Together AI-specific configuration
type TogetherConfig struct {
	BaseConfig
	APIKey string
}

// LMStudioConfig contains LM Studio-specific configuration
type LMStudioConfig struct {
	BaseConfig
	Endpoint string // Usually http://localhost:1234/v1
}

// MockConfig contains configuration for the mock LLM
type MockConfig struct {
	BaseConfig
	ResponsePrefix string
	DelayMs        int
}

// LoadOpenAIConfig loads OpenAI configuration from environment variables
func LoadOpenAIConfig() (*OpenAIConfig, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Get optional variables with defaults
	model := getEnvWithDefault("OPENAI_MODEL", "gpt-4o")
	org := os.Getenv("OPENAI_ORGANIZATION") // Optional
	temp, _ := strconv.ParseFloat(getEnvWithDefault("OPENAI_TEMPERATURE", "0.7"), 32)
	maxTokens, _ := strconv.Atoi(getEnvWithDefault("OPENAI_MAX_TOKENS", "1024"))

	return &OpenAIConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderOpenAI,
			Model:       model,
			Temperature: float32(temp),
			MaxTokens:   maxTokens,
		},
		APIKey:       apiKey,
		Organization: org,
	}, nil
}

// LoadAnthropicConfig loads Anthropic configuration from environment variables
func LoadAnthropicConfig() (*AnthropicConfig, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	model := getEnvWithDefault("ANTHROPIC_MODEL", "claude-3-opus-20240229")
	temp, _ := strconv.ParseFloat(getEnvWithDefault("ANTHROPIC_TEMPERATURE", "0.7"), 32)
	maxTokens, _ := strconv.Atoi(getEnvWithDefault("ANTHROPIC_MAX_TOKENS", "1024"))

	return &AnthropicConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderAnthropicAI,
			Model:       model,
			Temperature: float32(temp),
			MaxTokens:   maxTokens,
		},
		APIKey: apiKey,
	}, nil
}

// LoadOllamaConfig loads Ollama configuration from environment variables
func LoadOllamaConfig() (*OllamaConfig, error) {
	endpoint := getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434")
	model := getEnvWithDefault("OLLAMA_MODEL", "llama3")
	temp, _ := strconv.ParseFloat(getEnvWithDefault("OLLAMA_TEMPERATURE", "0.7"), 32)
	maxTokens, _ := strconv.Atoi(getEnvWithDefault("OLLAMA_MAX_TOKENS", "1024"))

	return &OllamaConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderOllama,
			Model:       model,
			Temperature: float32(temp),
			MaxTokens:   maxTokens,
		},
		Endpoint: endpoint,
	}, nil
}

// LoadTogetherConfig loads Together.ai configuration from environment variables
func LoadTogetherConfig() (*TogetherConfig, error) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	model := getEnvWithDefault("TOGETHER_MODEL", "togethercomputer/llama-3-8b")
	temp, _ := strconv.ParseFloat(getEnvWithDefault("TOGETHER_TEMPERATURE", "0.7"), 32)
	maxTokens, _ := strconv.Atoi(getEnvWithDefault("TOGETHER_MAX_TOKENS", "1024"))

	return &TogetherConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderTogetherAI,
			Model:       model,
			Temperature: float32(temp),
			MaxTokens:   maxTokens,
		},
		APIKey: apiKey,
	}, nil
}

// LoadLMStudioConfig loads LM Studio configuration from environment variables
func LoadLMStudioConfig() (*LMStudioConfig, error) {
	endpoint := getEnvWithDefault("LMSTUDIO_ENDPOINT", "http://localhost:1234/v1")
	model := getEnvWithDefault("LMSTUDIO_MODEL", "local-model")
	temp, _ := strconv.ParseFloat(getEnvWithDefault("LMSTUDIO_TEMPERATURE", "0.7"), 32)
	maxTokens, _ := strconv.Atoi(getEnvWithDefault("LMSTUDIO_MAX_TOKENS", "1024"))

	return &LMStudioConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderLMStudio,
			Model:       model,
			Temperature: float32(temp),
			MaxTokens:   maxTokens,
		},
		Endpoint: endpoint,
	}, nil
}

// LoadMockConfig creates a mock configuration for testing
func LoadMockConfig() *MockConfig {
	return &MockConfig{
		BaseConfig: BaseConfig{
			Provider:    ProviderMock,
			Model:       "mock-model",
			Temperature: 0.7,
			MaxTokens:   1024,
		},
		ResponsePrefix: "Thank you for asking me: ",
		DelayMs:        0, // No delay by default
	}
}

// LoadDefaultConfig loads the default LLM configuration based on available providers
func LoadDefaultConfig() (ProviderConfig, error) {
	// Try OpenAI first
	if config, err := LoadOpenAIConfig(); err == nil {
		return config, nil
	}

	// Try Anthropic next
	if config, err := LoadAnthropicConfig(); err == nil {
		return config, nil
	}

	// Try Together
	if config, err := LoadTogetherConfig(); err == nil {
		return config, nil
	}

	// Try local providers
	if config, err := LoadOllamaConfig(); err == nil {
		return config, nil
	}

	if config, err := LoadLMStudioConfig(); err == nil {
		return config, nil
	}

	// Fall back to mock for testing
	return LoadMockConfig(), nil
}

// Helper function to get environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// NewDefaultLLM creates a new LLM instance with the default configuration
func NewDefaultLLM(ctx context.Context) (LanguageModel, error) {
	config, err := LoadDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load default configuration: %w", err)
	}

	return NewLLM(ctx, config)
}
