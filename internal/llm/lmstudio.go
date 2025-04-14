package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LMStudioLLM is the LM Studio implementation of the LLM interface
type LMStudioLLM struct {
	// LM Studio exposes an OpenAI-compatible API
	client           *http.Client
	endpoint         string // Usually "http://localhost:1234/v1"
	model            string // Name of locally loaded model
	temperature      float32
	maxTokens        int
	timeoutSec       int     // Timeout in seconds for requests
	topP             float32 // Top-p sampling parameter
	presencePenalty  float32 // Presence penalty parameter
	frequencyPenalty float32 // Frequency penalty parameter
}

// LMStudioOption is a function that configures an LMStudioLLM
type LMStudioOption func(*LMStudioLLM)

// LMStudioRequest represents a request to the LM Studio API for completions
type LMStudioRequest struct {
	Model            string    `json:"model"`
	Prompt           string    `json:"prompt,omitempty"`
	Messages         []Message `json:"messages,omitempty"`
	Temperature      float32   `json:"temperature"`
	MaxTokens        int       `json:"max_tokens"`
	TopP             float32   `json:"top_p,omitempty"`
	FrequencyPenalty float32   `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32   `json:"presence_penalty,omitempty"`
	Stream           bool      `json:"stream,omitempty"`
}

// LMStudioChatResponse represents a response from the LM Studio chat API
type LMStudioChatResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Created           int64  `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	Choices           []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// LMStudioCompletionResponse represents a response from the LM Studio completions API
type LMStudioCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewLMStudioLLM creates a new LM Studio LLM with the specified options
func NewLMStudioLLM(endpoint string, options ...LMStudioOption) (*LMStudioLLM, error) {
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1" // Default endpoint
	}

	// Default values
	llm := &LMStudioLLM{
		client: &http.Client{
			Timeout: 60 * time.Second, // Default timeout
		},
		endpoint:         endpoint,
		model:            "local-model", // Default value - actual model is selected in LM Studio GUI
		temperature:      0.7,
		maxTokens:        1024,
		timeoutSec:       60,
		topP:             1.0,
		presencePenalty:  0.0,
		frequencyPenalty: 0.0,
	}

	// Apply options
	for _, option := range options {
		option(llm)
	}

	// Update client timeout based on timeoutSec setting
	llm.client.Timeout = time.Duration(llm.timeoutSec) * time.Second

	// Validate endpoint
	if !strings.HasPrefix(llm.endpoint, "http") {
		return nil, fmt.Errorf("invalid LM Studio endpoint: %s, must start with http or https", llm.endpoint)
	}

	return llm, nil
}

// WithLMStudioModel sets the model name for the LM Studio LLM
func WithLMStudioModel(model string) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.model = model
	}
}

// WithLMStudioTemperature sets the temperature for the LM Studio LLM
func WithLMStudioTemperature(temp float32) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.temperature = temp
	}
}

// WithLMStudioMaxTokens sets the max tokens for the LM Studio LLM
func WithLMStudioMaxTokens(maxTokens int) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.maxTokens = maxTokens
	}
}

// WithLMStudioTimeout sets the timeout for requests
func WithLMStudioTimeout(timeoutSec int) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.timeoutSec = timeoutSec
	}
}

// WithLMStudioTopP sets the top-p sampling parameter
func WithLMStudioTopP(topP float32) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.topP = topP
	}
}

// WithLMStudioPresencePenalty sets the presence penalty
func WithLMStudioPresencePenalty(penalty float32) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.presencePenalty = penalty
	}
}

// WithLMStudioFrequencyPenalty sets the frequency penalty
func WithLMStudioFrequencyPenalty(penalty float32) LMStudioOption {
	return func(l *LMStudioLLM) {
		l.frequencyPenalty = penalty
	}
}

// GenerateResponse generates a text response for a single prompt
func (l *LMStudioLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// LM Studio implements OpenAI-compatible API
	// For single prompt, we'll use the completions endpoint
	endpoint := fmt.Sprintf("%s/completions", l.endpoint)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	// Create request payload
	request := LMStudioRequest{
		Model:            l.model,
		Prompt:           prompt,
		Temperature:      l.temperature,
		MaxTokens:        l.maxTokens,
		TopP:             l.topP,
		FrequencyPenalty: l.frequencyPenalty,
		PresencePenalty:  l.presencePenalty,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to LM Studio: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LM Studio API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response LMStudioCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from choices
	if len(response.Choices) == 0 {
		return "", errors.New("invalid response")
	}

	return response.Choices[0].Text, nil
}

// GenerateChat generates a response based on a conversation history
func (l *LMStudioLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// For chat, we'll use the chat/completions endpoint
	endpoint := fmt.Sprintf("%s/chat/completions", l.endpoint)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue processing
	}

	// Create request payload
	request := LMStudioRequest{
		Model:            l.model,
		Messages:         messages,
		Temperature:      l.temperature,
		MaxTokens:        l.maxTokens,
		TopP:             l.topP,
		FrequencyPenalty: l.frequencyPenalty,
		PresencePenalty:  l.presencePenalty,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to LM Studio: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LM Studio API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response LMStudioChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w\nResponse body: %s", err, string(body))
	}

	// Extract text from choices
	if len(response.Choices) == 0 {
		return "", errors.New("invalid response")
	}

	return response.Choices[0].Message.Content, nil
}

// Initialize the factory function
func init() {
	newLMStudioFromConfig = createLMStudioFromConfig
}

// Implementation for creating an LMStudioLLM from config
func createLMStudioFromConfig(config ProviderConfig) (LanguageModel, error) {
	// Check if it's specifically an LMStudioConfig
	lmStudioConfig, ok := config.(*LMStudioConfig)
	if !ok {
		// Try to extract basic information from the generic config
		baseConfig, ok := config.(BaseConfig)
		if !ok {
			return nil, errors.New("invalid configuration type for LM Studio")
		}

		// Get endpoint from environment if not in config
		endpoint := getEnvWithDefault("LMSTUDIO_ENDPOINT", "http://localhost:1234/v1")

		return NewLMStudioLLM(
			endpoint,
			WithLMStudioModel(baseConfig.Model),
			WithLMStudioTemperature(baseConfig.Temperature),
			WithLMStudioMaxTokens(baseConfig.MaxTokens),
		)
	}

	// Use the specific LM Studio configuration
	options := []LMStudioOption{
		WithLMStudioModel(lmStudioConfig.Model),
		WithLMStudioTemperature(lmStudioConfig.Temperature),
		WithLMStudioMaxTokens(lmStudioConfig.MaxTokens),
	}

	return NewLMStudioLLM(lmStudioConfig.Endpoint, options...)
}
