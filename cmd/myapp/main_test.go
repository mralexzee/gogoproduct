package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

// mockLLMClient is a special flag to force the out-of-office mode for tests
var mockLLMClient bool = true

// Sleep duration between test messages to allow processing
const testSleepDuration = 2 * time.Second

// To support this test, RunCLIChatApp must be defined in main.go or another imported package.
// It should accept io.Reader and io.Writer for input/output redirection.
func TestChatApp_OutOfOfficeFallback(t *testing.T) {
	// Create a pipe for writing messages with delays
	pipeReader, pipeWriter := io.Pipe()
	out := &bytes.Buffer{}

	// Start the chat app in a goroutine
	go func() {
		err := RunCLIChatApp(pipeReader, out)
		if err != nil {
			t.Errorf("app failed: %v", err)
		}
	}()

	// Wait for the app to initialize
	time.Sleep(500 * time.Millisecond)

	// Send messages with delay to allow timeout responses to be generated
	messages := []string{"Hello", "bye", "exit()"}
	for _, msg := range messages {
		// Write the message
		_, err := pipeWriter.Write([]byte(msg + "\n"))
		if err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}

		// Wait for the out-of-office timeout to trigger
		time.Sleep(testSleepDuration)
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
		"Andy: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.",
		"User: bye",
		"Andy: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.",
		"Goodbye!",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected text: %q", want)
		}
	}
}
