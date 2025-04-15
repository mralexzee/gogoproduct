package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestEchoLLMInstant tests the chat application with an instant-responding EchoLLM
func TestEchoLLMInstant(t *testing.T) {
	// Set environment variables for the LLM
	os.Setenv("LLM_TYPE", "echo")
	os.Setenv("LLM_DELAY", "0")
	defer func() {
		os.Unsetenv("LLM_TYPE")
		os.Unsetenv("LLM_DELAY")
	}()

	// Create a pipe for input/output simulation
	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	// Buffer to collect output
	out := new(bytes.Buffer)

	// Run the chat application in a goroutine
	go func() {
		err := RunCLIChatApp(pipeReader, out)
		if err != nil {
			t.Errorf("RunCLIChatApp returned error: %v", err)
		}
	}()

	// Wait for the app to initialize
	time.Sleep(500 * time.Millisecond)

	// Send messages
	messages := []string{"Hello", "What's the status of our project?", "exit()"}
	for _, msg := range messages {
		// Write the message
		_, err := pipeWriter.Write([]byte(msg + "\n"))
		if err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}

		// Wait to ensure response is captured
		time.Sleep(200 * time.Millisecond)
	}

	// Close the writer to signal EOF
	pipeWriter.Close()

	// Wait for final processing
	time.Sleep(500 * time.Millisecond)

	// Check the output for expected patterns
	output := out.String()
	t.Logf("Output:\n%s", output)

	expected := []string{
		"Welcome to the Enhanced Chat Interface!",
		"User: Hello",
		"Andy: ECHO: You said: Hello",
		"User: What's the status of our project?",
		"Andy: ECHO: You said: What's the status of our project?",
		"Goodbye!",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected text: %q", want)
		}
	}
}
