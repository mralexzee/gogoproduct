package logging

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LogLevel represents the severity of a log message
type LogLevel int

// Log levels
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts our LogLevel to slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger is a simple wrapper around slog
type Logger struct {
	Logger *slog.Logger // Capitalized for direct access
	writer io.Writer
}

// Global singleton logger instance
var defaultLogger *Logger

// Init initializes the default logger
func Init(logger *Logger) {
	defaultLogger = logger
}

// Get returns the default logger
func Get() *Logger {
	if defaultLogger == nil {
		// Default to console if not initialized
		defaultLogger = Console()
	}
	return defaultLogger
}

// customHandler implements a handler with our custom format
type customHandler struct {
	level     slog.Level
	addSource bool
	w         io.Writer
}

// Enabled reports whether the handler handles records at the given level
func (h *customHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats a log record as specified and writes it to the output
func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006-01-02 15:04:05Z07:00")
	level := r.Level.String()

	// Get source info if enabled
	var file string
	var line int
	if h.addSource {
		if r.PC != 0 {
			// Get the caller's file information from the PC
			frames := runtime.CallersFrames([]uintptr{r.PC - 1})
			frame, _ := frames.Next()

			// Extract just the filename, not the full path
			file = filepath.Base(frame.File)
			line = frame.Line
		} else {
			file = "???"
			line = 0
		}
	}

	// Extract attributes for structured logging
	attrs := make([]string, 0)
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "" && attr.Value.String() != "" {
			attrs = append(attrs, fmt.Sprintf("%s=%s", attr.Key, attr.Value.String()))
		}
		return true
	})

	// Build the log message
	messageWithAttrs := r.Message
	if len(attrs) > 0 {
		messageWithAttrs += " [" + strings.Join(attrs, ", ") + "]"
	}

	var buf []byte
	if h.addSource {
		lineStr := fmt.Sprintf("%d", line)
		buf = []byte(timeStr + " " + level + " " + file + ":" + lineStr + " " + messageWithAttrs + "\n")
	} else {
		buf = []byte(timeStr + " " + level + " " + messageWithAttrs + "\n")
	}

	_, err := h.w.Write(buf)
	return err
}

// WithAttrs returns a new handler whose attributes consist of h's attributes
// followed by attrs
func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, we're ignoring attributes in this implementation
	return h
}

// WithGroup returns a new handler with the given group appended to the receiver's
// existing groups
func (h *customHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we're ignoring groups in this implementation
	return h
}

// File creates a new file logger
func File(filename string, append bool) *Logger {
	var flag int
	if append {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	file, err := os.OpenFile(filename, flag, 0666)
	if err != nil {
		// Fall back to stderr if file can't be opened
		return &Logger{
			Logger: slog.New(&customHandler{
				level:     slog.LevelInfo,
				addSource: true,
				w:         os.Stderr,
			}),
			writer: os.Stderr,
		}
	}

	handler := &customHandler{
		level:     slog.LevelDebug,
		addSource: true,
		w:         file,
	}

	return &Logger{
		Logger: slog.New(handler),
		writer: file,
	}
}

// Console creates a logger that writes to stderr
func Console() *Logger {
	handler := &customHandler{
		level:     slog.LevelInfo,
		addSource: true,
		w:         os.Stderr,
	}

	return &Logger{
		Logger: slog.New(handler),
		writer: os.Stderr,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}

// Close closes the logger if needed (e.g., file handle)
func (l *Logger) Close() error {
	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// DevNull creates a logger that discards all output
func DevNull() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewTextHandler(ioutil.Discard, nil)),
		writer: ioutil.Discard,
	}
}
