package llm

import "time"

// Message represents a single message in a conversation
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// RequestOptions contains common parameters for LLM requests
type RequestOptions struct {
	Temperature      float32 // Controls randomness (0.0 to 1.0)
	MaxTokens        int     // Maximum tokens to generate
	TopP             float32 // Controls diversity via nucleus sampling
	FrequencyPenalty float32 // Reduces repetition of token sequences
	PresencePenalty  float32 // Reduces repetition of topics
}

// DefaultRequestOptions provides sensible defaults for request options
var DefaultRequestOptions = RequestOptions{
	Temperature:      0.7,
	MaxTokens:        1024,
	TopP:             1.0,
	FrequencyPenalty: 0.0,
	PresencePenalty:  0.0,
}

// ProviderConfig is the interface that all provider configurations must implement
type ProviderConfig interface {
	GetProvider() string
	GetModel() string
}

// BaseConfig contains common configuration for all LLM providers
type BaseConfig struct {
	Provider    string
	Model       string
	Temperature float32
	MaxTokens   int
}

// GetProvider returns the provider name
func (c BaseConfig) GetProvider() string {
	return c.Provider
}

// GetModel returns the model name
func (c BaseConfig) GetModel() string {
	return c.Model
}

// Usage contains information about token usage for an LLM request
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Response encapsulates a standard response from an LLM
type Response struct {
	Text         string
	Usage        Usage
	FinishReason string
	CreatedAt    time.Time
}
