package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// EchoLLM implements a simple echo-based LLM for testing
// It responds with "ECHO: You said: <input>" after a configurable delay
type EchoLLM struct {
	delay time.Duration
}

// NewEchoLLM creates a new EchoLLM with the specified delay in seconds
func NewEchoLLM(delaySec int) *EchoLLM {
	return &EchoLLM{
		delay: time.Duration(delaySec) * time.Second,
	}
}

// GenerateResponse implements the LLM interface for a single prompt
func (e *EchoLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Wait for the configured delay
	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Return the echo response with the ECHO prefix and expected format
	return fmt.Sprintf("ECHO: You said: %s", prompt), nil
}

// GenerateChat implements the LLM interface for a conversation
func (e *EchoLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// Extract the last user message as the prompt
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	// Generate response using the prompt
	return e.GenerateResponse(ctx, lastUserMessage)
}

// ExceptionLLM implements an LLM that always throws an exception after a delay
type ExceptionLLM struct {
	delay time.Duration
}

// NewExceptionLLM creates a new ExceptionLLM with the specified delay in seconds
func NewExceptionLLM(delaySec int) *ExceptionLLM {
	return &ExceptionLLM{
		delay: time.Duration(delaySec) * time.Second,
	}
}

// GenerateResponse implements the LLM interface for a single prompt but always returns an error
func (e *ExceptionLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Wait for the configured delay
	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Return an error
	return "", errors.New("Exception occurred in this mock LLM, as expected")
}

// GenerateChat implements the LLM interface for a conversation but always returns an error
func (e *ExceptionLLM) GenerateChat(ctx context.Context, messages []Message) (string, error) {
	// Just use the same error handling as GenerateResponse
	return e.GenerateResponse(ctx, "")
}

// Configuration structs for factory initialization
type EchoConfig struct {
	BaseConfig
	DelaySeconds int
}

func (c *EchoConfig) GetProvider() string {
	return ProviderEcho
}

type ExceptionConfig struct {
	BaseConfig
	DelaySeconds int
}

func (c *ExceptionConfig) GetProvider() string {
	return ProviderException
}

// Initialize the factory functions
func init() {
	newEchoFromConfig = createEchoFromConfig
	newExceptionFromConfig = createExceptionFromConfig
}

// Implementation for creating EchoLLM from config
func createEchoFromConfig(config ProviderConfig) (LanguageModel, error) {
	delaySec := 0

	// Check if the config has a delay specified
	if echoConfig, ok := config.(*EchoConfig); ok {
		delaySec = echoConfig.DelaySeconds
	} else {
		// Check for environment variable
		if envDelay := os.Getenv("LLM_DELAY"); envDelay != "" {
			if delay, err := strconv.Atoi(envDelay); err == nil {
				delaySec = delay
			}
		}
	}

	// Ensure delay is within bounds
	if delaySec < 0 {
		delaySec = 0
	} else if delaySec > 10 {
		delaySec = 10
	}

	return NewEchoLLM(delaySec), nil
}

// Implementation for creating ExceptionLLM from config
func createExceptionFromConfig(config ProviderConfig) (LanguageModel, error) {
	delaySec := 0

	// Check if the config has a delay specified
	if exceptionConfig, ok := config.(*ExceptionConfig); ok {
		delaySec = exceptionConfig.DelaySeconds
	} else {
		// Check for environment variable
		if envDelay := os.Getenv("LLM_DELAY"); envDelay != "" {
			if delay, err := strconv.Atoi(envDelay); err == nil {
				delaySec = delay
			}
		}
	}

	// Ensure delay is within bounds
	if delaySec < 0 {
		delaySec = 0
	} else if delaySec > 10 {
		delaySec = 10
	}

	return NewExceptionLLM(delaySec), nil
}
