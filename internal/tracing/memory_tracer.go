package tracing

import (
	"bytes"
	"sync"
)

// MemoryTracer is a tracer that stores events in memory for testing
type MemoryTracer struct {
	buffer *bytes.Buffer
	level  Level
	mu     sync.Mutex
}

// NewMemoryTracer creates a new MemoryTracer for testing
func NewMemoryTracer() *EnhancedTracer {
	memTracer := &MemoryTracer{
		buffer: &bytes.Buffer{},
		level:  LevelDebug, // Default to debug level for tests
	}
	return NewEnhancedTracer(memTracer, "test")
}

// Trace writes the event to the in-memory buffer
func (t *MemoryTracer) Trace(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Level > t.level {
		return nil
	}

	line := formatEventLine(event)
	t.buffer.WriteString(line)
	t.buffer.WriteString("\n")
	return nil
}

// GetOutput returns the current buffer contents
func (t *MemoryTracer) GetOutput() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.buffer.String()
}

// Clear clears the buffer contents
func (t *MemoryTracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.buffer.Reset()
}

// Flush does nothing for memory tracer (no buffering needed)
func (t *MemoryTracer) Flush() error {
	return nil
}

// Close does nothing for memory tracer
func (t *MemoryTracer) Close() error {
	return nil
}

// SetLevel sets the minimum level of events to trace
func (t *MemoryTracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.level = level
}
