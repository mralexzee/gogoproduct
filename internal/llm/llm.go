package llm

import (
	"context"
	"errors"
)

// LanguageModel defines the core interface that all language model implementations must satisfy
type LanguageModel interface {
	// GenerateResponse generates a text response for a single prompt
	GenerateResponse(ctx context.Context, prompt string) (string, error)

	// GenerateChat generates a response based on a conversation history
	GenerateChat(ctx context.Context, messages []Message) (string, error)
}

// StreamingLLM is an optional interface for LLMs that support streaming responses
type StreamingLLM interface {
	// StreamResponse streams a response token by token for a single prompt
	StreamResponse(ctx context.Context, prompt string) (chan StreamToken, error)

	// StreamChat streams a response token by token for a conversation history
	StreamChat(ctx context.Context, messages []Message) (chan StreamToken, error)
}

// StreamToken represents a token in a streaming response
type StreamToken struct {
	Text  string
	Error error
	Done  bool
}

// Common error types for LLM operations
var (
	ErrAPIKeyMissing   = errors.New("API key missing")
	ErrRateLimited     = errors.New("rate limited by provider")
	ErrInvalidResponse = errors.New("invalid response from LLM")
	ErrContextTooLarge = errors.New("context size exceeds model limits")
	ErrProviderError   = errors.New("provider returned an error")
)
