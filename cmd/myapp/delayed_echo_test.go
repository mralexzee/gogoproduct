package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestEchoLLMDelayed tests the chat application with a delayed-responding EchoLLM
func TestEchoLLMDelayed(t *testing.T) {
	// Set environment variables for the LLM
	os.Setenv("LLM_TYPE", "echo")
	os.Setenv("LLM_DELAY", "2")
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
	messages := []string{"Hello", "Tell me about the roadmap", "exit()"}
	for _, msg := range messages {
		// Write the message
		_, err := pipeWriter.Write([]byte(msg + "\n"))
		if err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}

		// Wait for the delayed response (a bit more than the 2-second delay)
		time.Sleep(3 * time.Second)
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
		"User: Tell me about the roadmap",
		"Andy: ECHO: You said: Tell me about the roadmap",
		"Goodbye!",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected text: %q", want)
		}
	}

	// Also verify no out-of-office message appears despite the delay
	unexpected := "I'm out of office today"
	if strings.Contains(output, unexpected) {
		t.Errorf("output unexpectedly contains text: %q", unexpected)
	}
}
