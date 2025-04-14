package tracing

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

// EnhancedTracer provides convenient logging methods over the base Tracer interface
type EnhancedTracer struct {
	tracer   Tracer
	sourceID string
}

// NewEnhancedTracer creates a new enhanced tracer wrapping a base tracer
func NewEnhancedTracer(baseTracer Tracer, sourceID string) *EnhancedTracer {
	return &EnhancedTracer{
		tracer:   baseTracer,
		sourceID: sourceID,
	}
}

// Info logs an informational message
func (t *EnhancedTracer) Info(format string, args ...interface{}) {
	t.log(LevelInfo, format, args...)
}

// Debug logs a debug message
func (t *EnhancedTracer) Debug(format string, args ...interface{}) {
	t.log(LevelDebug, format, args...)
}

// Error logs an error message
func (t *EnhancedTracer) Error(format string, args ...interface{}) {
	t.log(LevelError, format, args...)
}

// Warning logs a warning message
func (t *EnhancedTracer) Warning(format string, args ...interface{}) {
	t.log(LevelWarning, format, args...)
}

// Verbose logs a verbose message
func (t *EnhancedTracer) Verbose(format string, args ...interface{}) {
	t.log(LevelVerbose, format, args...)
}

// log creates and sends a trace event with the given level and message
func (t *EnhancedTracer) log(level Level, format string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(2) // Get caller info for better tracing
	event := Event{
		Timestamp: time.Now(),
		Component: "system", // Default component
		Operation: "log",    // Default operation for logging
		Level:     level,
		SourceID:  t.sourceID,
		Message:   fmt.Sprintf(format, args...),
		Metadata: map[string]interface{}{
			"file": filepath.Base(file),
			"line": line,
		},
	}
	_ = t.tracer.Trace(event)
}

// Close closes the underlying tracer
func (t *EnhancedTracer) Close() error {
	return t.tracer.Close()
}

// Flush flushes the underlying tracer
func (t *EnhancedTracer) Flush() error {
	return t.tracer.Flush()
}

// Trace passes the event to the underlying tracer
func (t *EnhancedTracer) Trace(event Event) error {
	return t.tracer.Trace(event)
}

// SetLevel sets the level of the underlying tracer
func (t *EnhancedTracer) SetLevel(level Level) {
	t.tracer.SetLevel(level)
}

// CreateFileTracer creates a new enhanced tracer with a file backend
// This is a convenience function to create a properly configured file tracer
func CreateFileTracer(filePath string, flushInterval time.Duration, bufferSize int) (*EnhancedTracer, error) {
	// Create options for the file tracer
	options := FileTracerOptions{
		FilePath:      filePath,
		Append:        true,
		BufferSize:    bufferSize,
		FlushInterval: flushInterval,
		Level:         LevelDebug, // Default to debug level
	}

	// Create the file tracer
	fileTracer, err := NewFileTracer(options)
	if err != nil {
		return nil, err
	}

	// Create and return the enhanced tracer
	return NewEnhancedTracer(fileTracer, "system"), nil
}
