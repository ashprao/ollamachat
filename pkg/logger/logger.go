package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger with the specified level
func NewLogger(level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{slog.New(handler)}
}

// WithComponent returns a new logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{l.With("component", component)}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{l.With(args...)}
}
