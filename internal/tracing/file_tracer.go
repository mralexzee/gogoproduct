package tracing

import (
	"fmt"
	"os"
	"time"
)

// DefaultBufferSize is the default size in bytes before flushing (4KB)
const DefaultBufferSize = 4 * 1024

// DefaultFlushInterval is the default time interval between flushes
const DefaultFlushInterval = 5 * time.Second

// DefaultLogFile is the default path for trace log files
const DefaultLogFile = "./trace.log"

// NewDefaultFileTracer creates a file tracer with default settings
// that appends to "./trace.log" with a 4KB buffer and 5 second flush interval
func NewDefaultFileTracer() (*FileTracer, error) {
	return NewFileTracer(DefaultFileTracerOptions())
}

// FileTracerOptionFunc is a function that configures a FileTracerOptions
type FileTracerOptionFunc func(*FileTracerOptions)

// WithFilePath sets the file path for the tracer
func WithFilePath(path string) FileTracerOptionFunc {
	return func(o *FileTracerOptions) {
		o.FilePath = path
	}
}

// WithAppend sets whether to append to the file or replace it
func WithAppend(append bool) FileTracerOptionFunc {
	return func(o *FileTracerOptions) {
		o.Append = append
	}
}

// WithBufferSize sets the buffer size in bytes
func WithBufferSize(size int) FileTracerOptionFunc {
	return func(o *FileTracerOptions) {
		o.BufferSize = size
	}
}

// WithFlushInterval sets the flush interval
func WithFlushInterval(interval time.Duration) FileTracerOptionFunc {
	return func(o *FileTracerOptions) {
		o.FlushInterval = interval
	}
}

// WithLevel sets the trace level
func WithLevel(level Level) FileTracerOptionFunc {
	return func(o *FileTracerOptions) {
		o.Level = level
	}
}

// NewFileTracerWithOptions creates a file tracer with the provided options
func NewFileTracerWithOptions(opts ...FileTracerOptionFunc) (*FileTracer, error) {
	options := DefaultFileTracerOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return NewFileTracer(options)
}

// EnsureDirectoryExists makes sure the directory for the specified file exists
func EnsureDirectoryExists(filePath string) error {
	dir := ""
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			dir = filePath[:i]
			break
		}
	}

	// If there's no directory component, we're done
	if dir == "" {
		return nil
	}

	// Create the directory with all parents if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for trace file: %w", err)
	}

	return nil
}
