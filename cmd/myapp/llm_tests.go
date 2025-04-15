package main

// This file contains tests for various mock LLM implementations

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// Test_ChatApp_EchoLLM_Instant tests the chat application with an instant-responding EchoLLM
func Test_ChatApp_EchoLLM_Instant(t *testing.T) {
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
		"Andy: You said: Hello",
		"User: What's the status of our project?",
		"Andy: You said: What's the status of our project?",
		"Goodbye!",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected text: %q", want)
		}
	}
}

// Test_ChatApp_EchoLLM_Delayed tests the chat application with a delayed-responding EchoLLM
func Test_ChatApp_EchoLLM_Delayed(t *testing.T) {
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
		time.Sleep(2500 * time.Millisecond)
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
		"Andy: You said: Hello",
		"User: Tell me about the roadmap",
		"Andy: You said: Tell me about the roadmap",
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

// Test_ChatApp_ExceptionLLM tests the chat application with an ExceptionLLM that simulates errors
func Test_ChatApp_ExceptionLLM(t *testing.T) {
	// Set environment variables for the LLM
	os.Setenv("LLM_TYPE", "exception")
	os.Setenv("LLM_DELAY", "1")
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

	// Send messages with delay to allow fallback responses to be generated
	messages := []string{"Hello", "What's going on?", "exit()"}
	for _, msg := range messages {
		// Write the message
		_, err := pipeWriter.Write([]byte(msg + "\n"))
		if err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}

		// Wait for the exception response & fallback (a bit more than the 1-second delay)
		time.Sleep(1500 * time.Millisecond)
	}

	// Close the writer to signal EOF
	pipeWriter.Close()

	// Wait for final processing
	time.Sleep(1000 * time.Millisecond)

	// Check the output for expected patterns
	output := out.String()
	t.Logf("Output:\n%s", output)

	expected := []string{
		"Welcome to the Enhanced Chat Interface!",
		"User: Hello",
		"Andy: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.",
		"User: What's going on?",
		"Andy: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.",
		"Goodbye!",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected text: %q", want)
		}
	}

	// Verify that there are no echo responses (which would indicate a failure in our error handling)
	unexpected := "You said:"
	if strings.Contains(output, unexpected) {
		t.Errorf("output unexpectedly contains text: %q", unexpected)
	}
}
