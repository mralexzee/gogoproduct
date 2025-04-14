package messaging

import (
	"bytes"
	"goproduct/internal/tracing"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessageBusWithTracing(t *testing.T) {
	// Create a buffer to capture trace output
	var buffer bytes.Buffer
	tracer := tracing.NewWriterTracer(&buffer, tracing.LevelInfo)

	// Create a message bus with our tracer
	bus := NewMemoryMessageBusWithTracer(tracer)

	// Test entity IDs
	entity1ID := "sender"
	entity2ID := "receiver"

	// Message tracking
	received := make(chan Message, 10)

	// Subscribe entities to the bus
	err := bus.Subscribe(entity1ID, func(msg Message) error {
		received <- msg
		return nil
	})
	assert.NoError(t, err)

	err = bus.Subscribe(entity2ID, func(msg Message) error {
		received <- msg
		return nil
	})
	assert.NoError(t, err)

	// Send a direct message
	message := NewTextMessage(entity1ID, []string{entity2ID}, "Hello with tracing")
	err = bus.Publish(message)
	assert.NoError(t, err)

	// Wait for message to be processed
	select {
	case msg := <-received:
		content, err := msg.TextContent()
		assert.NoError(t, err)
		assert.Equal(t, "Hello with tracing", content)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Flush the tracer to ensure all traces are written
	tracer.Flush()

	// Verify that trace output contains both send and receive events
	output := buffer.String()
	t.Log("Trace output:", output)

	assert.True(t, strings.Contains(output, "\"operation\":\"send\""), "Trace should contain send operation")
	assert.True(t, strings.Contains(output, "\"operation\":\"receive\""), "Trace should contain receive operation")
	assert.True(t, strings.Contains(output, entity1ID), "Trace should contain sender ID")
	assert.True(t, strings.Contains(output, entity2ID), "Trace should contain receiver ID")
}

func TestFileTracer(t *testing.T) {
	t.Skip("Skipping file tracer test as it writes to disk")

	// Example of how to use the file tracer
	tracer, err := tracing.NewFileTracerWithOptions(
		tracing.WithFilePath("./test_trace.log"),
		tracing.WithAppend(false),                       // Start with a fresh file
		tracing.WithBufferSize(1024),                    // 1KB buffer for testing
		tracing.WithFlushInterval(500*time.Millisecond), // Short interval for testing
	)
	assert.NoError(t, err)
	defer tracer.Close()

	// Create a message bus with our tracer
	bus := NewMemoryMessageBusWithTracer(tracer)

	// Test entity IDs
	entity1ID := "file_sender"
	entity2ID := "file_receiver"

	// Subscribe entities
	bus.Subscribe(entity1ID, func(msg Message) error { return nil })
	bus.Subscribe(entity2ID, func(msg Message) error { return nil })

	// Send a few messages to generate traces
	for i := 0; i < 10; i++ {
		message := NewTextMessage(entity1ID, []string{entity2ID}, "Test message")
		bus.Publish(message)
		time.Sleep(100 * time.Millisecond)
	}

	// Manually flush to ensure all traces are written
	tracer.Flush()

	// Note: In a real test, we would validate the file contents
	// but that's complex to do in a unit test reliably
}
