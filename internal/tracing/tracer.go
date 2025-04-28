package tracing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Component identifies the system component generating the trace
type Component string

const (
	// ComponentMessaging identifies the messaging system
	ComponentMessaging Component = "messaging"
	// ComponentMemory identifies the knowledge system
	ComponentMemory Component = "knowledge"
	// ComponentEntity identifies the entity system
	ComponentEntity Component = "entity"
	// ComponentAgent identifies the agent system
	ComponentAgent Component = "agent"
)

// Operation identifies the type of operation being traced
type Operation string

const (
	// OperationSend identifies a send operation
	OperationSend Operation = "send"
	// OperationReceive identifies a receive operation
	OperationReceive Operation = "receive"
	// OperationCreate identifies a creation operation
	OperationCreate Operation = "create"
	// OperationDelete identifies a deletion operation
	OperationDelete Operation = "delete"
	// OperationUpdate identifies an update operation
	OperationUpdate Operation = "update"
	// OperationJoin identifies a join operation (e.g., joining a group)
	OperationJoin Operation = "join"
	// OperationLeave identifies a leave operation (e.g., leaving a group)
	OperationLeave Operation = "leave"
)

// Level defines the verbosity level of tracing
type Level int

const (
	// LevelError only traces errors
	LevelError Level = iota
	// LevelWarning traces warnings and errors
	LevelWarning
	// LevelInfo traces general information, warnings, and errors
	LevelInfo
	// LevelDebug traces detailed information for debugging
	LevelDebug
	// LevelVerbose traces everything
	LevelVerbose
)

// Event represents a traceable event
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Component Component              `json:"component"`
	Operation Operation              `json:"operation"`
	Level     Level                  `json:"level"`
	SourceID  string                 `json:"source_id,omitempty"`
	TargetID  string                 `json:"target_id,omitempty"`
	ObjectID  string                 `json:"object_id,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Tracer defines the interface for system tracing
type Tracer interface {
	// Trace records a trace event
	Trace(event Event) error

	// Flush forces any buffered data to be written
	Flush() error

	// Close flushes all data and closes the tracer
	Close() error

	// SetLevel sets the minimum level of events to trace
	SetLevel(level Level)
}

// NoopTracer is a tracer that does nothing
type NoopTracer struct{}

// Trace does nothing and returns nil
func (t *NoopTracer) Trace(event Event) error {
	return nil
}

// Flush does nothing and returns nil
func (t *NoopTracer) Flush() error {
	return nil
}

// Close does nothing and returns nil
func (t *NoopTracer) Close() error {
	return nil
}

// SetLevel does nothing for NoopTracer
func (t *NoopTracer) SetLevel(level Level) {}

// NewNoopTracer creates a new NoopTracer
func NewNoopTracer() *NoopTracer {
	return &NoopTracer{}
}

// ConsoleTracer outputs trace events to stdout
type ConsoleTracer struct {
	level Level
	mu    sync.Mutex
}

// Trace writes the event to stdout if it meets the level threshold
func (t *ConsoleTracer) Trace(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Level > t.level {
		return nil
	}

	line := formatEventLine(event)
	_, err := fmt.Fprintln(os.Stdout, line)
	return err
}

// Flush does nothing for console tracer (no buffering)
func (t *ConsoleTracer) Flush() error {
	return nil
}

// Close does nothing for console tracer
func (t *ConsoleTracer) Close() error {
	return nil
}

// SetLevel sets the minimum level of events to trace
func (t *ConsoleTracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.level = level
}

// NewConsoleTracer creates a new ConsoleTracer
func NewConsoleTracer(level Level) *ConsoleTracer {
	return &ConsoleTracer{
		level: level,
	}
}

// FileTracer outputs trace events to a file with buffering
type FileTracer struct {
	file          *os.File
	buffer        *bytes.Buffer
	bufferSize    int
	flushInterval time.Duration
	timer         *time.Timer
	level         Level
	mu            sync.Mutex
}

// FileTracerOptions contains options for creating a FileTracer
type FileTracerOptions struct {
	FilePath      string
	Append        bool
	BufferSize    int
	FlushInterval time.Duration
	Level         Level
}

// DefaultFileTracerOptions returns the default options for FileTracer
func DefaultFileTracerOptions() FileTracerOptions {
	return FileTracerOptions{
		FilePath:      "trace.log",
		Append:        true,
		BufferSize:    4096, // 4KB
		FlushInterval: 5 * time.Second,
		Level:         LevelInfo,
	}
}

// NewFileTracer creates a new FileTracer
func NewFileTracer(options FileTracerOptions) (*FileTracer, error) {
	flag := os.O_CREATE | os.O_WRONLY
	if options.Append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(options.FilePath, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}

	tracer := &FileTracer{
		file:          file,
		buffer:        bytes.NewBuffer(make([]byte, 0, options.BufferSize)),
		bufferSize:    options.BufferSize,
		flushInterval: options.FlushInterval,
		level:         options.Level,
	}

	// Start the flush timer
	tracer.resetTimer()

	return tracer, nil
}

// resetTimer resets the flush timer
func (t *FileTracer) resetTimer() {
	if t.timer != nil {
		t.timer.Stop()
	}

	t.timer = time.AfterFunc(t.flushInterval, func() {
		_ = t.Flush()
		t.resetTimer()
	})
}

// Trace writes the event to the buffer if it meets the level threshold
func (t *FileTracer) Trace(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Level > t.level {
		return nil
	}

	line := formatEventLine(event)
	_, err := t.buffer.WriteString(line + "\n")
	if err != nil {
		return err
	}

	// Flush if buffer exceeds size threshold
	if t.buffer.Len() >= t.bufferSize {
		return t.flushLocked()
	}

	return nil
}

// flushLocked flushes the buffer to the file (assumes lock is already held)
func (t *FileTracer) flushLocked() error {
	if t.buffer.Len() > 0 {
		_, err := t.buffer.WriteTo(t.file)
		return err
	}
	return nil
}

// Flush forces the buffer to be written to the file
func (t *FileTracer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.flushLocked()
}

// Close flushes the buffer and closes the file
func (t *FileTracer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}

	if err := t.flushLocked(); err != nil {
		return err
	}

	return t.file.Close()
}

// SetLevel sets the minimum level of events to trace
func (t *FileTracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.level = level
}

// MultiTracer sends events to multiple tracers
type MultiTracer struct {
	tracers []Tracer
	level   Level
	mu      sync.Mutex
}

// NewMultiTracer creates a new MultiTracer
func NewMultiTracer(tracers ...Tracer) *MultiTracer {
	return &MultiTracer{
		tracers: tracers,
		level:   LevelInfo,
	}
}

// Trace sends the event to all tracers
func (t *MultiTracer) Trace(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Level > t.level {
		return nil
	}

	var lastErr error
	for _, tracer := range t.tracers {
		if err := tracer.Trace(event); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Flush flushes all tracers
func (t *MultiTracer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lastErr error
	for _, tracer := range t.tracers {
		if err := tracer.Flush(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Close closes all tracers
func (t *MultiTracer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lastErr error
	for _, tracer := range t.tracers {
		if err := tracer.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// SetLevel sets the minimum level of events to trace
func (t *MultiTracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.level = level
	for _, tracer := range t.tracers {
		tracer.SetLevel(level)
	}
}

// WriterTracer sends trace events to an io.Writer
type WriterTracer struct {
	writer io.Writer
	level  Level
	mu     sync.Mutex
}

// NewWriterTracer creates a new WriterTracer
func NewWriterTracer(writer io.Writer, level Level) *WriterTracer {
	return &WriterTracer{
		writer: writer,
		level:  level,
	}
}

// Trace writes the event to the io.Writer
func (t *WriterTracer) Trace(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if event.Level > t.level {
		return nil
	}

	line := formatEventLine(event)
	_, err := fmt.Fprintln(t.writer, line)
	return err
}

// Flush does nothing unless the writer is a Flusher
func (t *WriterTracer) Flush() error {
	if flusher, ok := t.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// Close does nothing unless the writer is a Closer
func (t *WriterTracer) Close() error {
	if closer, ok := t.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// SetLevel sets the minimum level of events to trace
func (t *WriterTracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.level = level
}

// formatEventLine formats an event as a single line of ASCII text
func formatEventLine(event Event) string {
	// First create a simplified version of the event for serialization
	simplified := struct {
		Timestamp string                 `json:"timestamp"`
		Component string                 `json:"component"`
		Operation string                 `json:"operation"`
		Level     int                    `json:"level"`
		SourceID  string                 `json:"source_id,omitempty"`
		TargetID  string                 `json:"target_id,omitempty"`
		ObjectID  string                 `json:"object_id,omitempty"`
		Message   string                 `json:"message,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}{
		Timestamp: event.Timestamp.Format(time.RFC3339Nano),
		Component: string(event.Component),
		Operation: string(event.Operation),
		Level:     int(event.Level),
		SourceID:  sanitizeString(event.SourceID),
		TargetID:  sanitizeString(event.TargetID),
		ObjectID:  sanitizeString(event.ObjectID),
		Message:   sanitizeString(event.Message),
		Metadata:  sanitizeMetadata(event.Metadata),
	}

	// Serialize to JSON
	data, err := json.Marshal(simplified)
	if err != nil {
		// Fallback if JSON serialization fails
		return fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s|%s",
			simplified.Timestamp,
			simplified.Component,
			simplified.Operation,
			simplified.Level,
			simplified.SourceID,
			simplified.TargetID,
			simplified.ObjectID,
			simplified.Message)
	}

	return sanitizeString(string(data))
}

// sanitizeString removes unsafe characters and ensures output is ASCII only
func sanitizeString(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		// Only allow printable ASCII characters (32-126)
		if r >= 32 && r <= 126 {
			result.WriteRune(r)
		} else if unicode.IsSpace(r) {
			// Replace all whitespace with a single space
			result.WriteRune(' ')
		}
		// Skip all other characters
	}

	return result.String()
}

// sanitizeMetadata recursively sanitizes all string values in the metadata
func sanitizeMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return nil
	}

	result := make(map[string]interface{}, len(metadata))

	for k, v := range metadata {
		key := sanitizeString(k)

		switch val := v.(type) {
		case string:
			result[key] = sanitizeString(val)
		case map[string]interface{}:
			result[key] = sanitizeMetadata(val)
		case []interface{}:
			sanitized := make([]interface{}, 0, len(val))
			for _, item := range val {
				if str, ok := item.(string); ok {
					sanitized = append(sanitized, sanitizeString(str))
				} else {
					sanitized = append(sanitized, item)
				}
			}
			result[key] = sanitized
		default:
			result[key] = val
		}
	}

	return result
}
