// Package stdlog provides a logger implementation using the standard library's slog package
// used as adapter for logging interfaces.
package stdlog

import (
	"fmt"
	"io"
	"log/slog"
)

// SlogLogger is the implementation of Logger using slog.
type SlogLogger struct {
	logger *slog.Logger
}

// NewLogger creates a new SlogLogger instance.
func NewLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{logger: logger}
}

// NewSlogLogger creates a new slog.Logger instance with the specified writer and format.
func NewSlogLogger(w io.Writer, isText bool) *slog.Logger {
	var handler slog.Handler
	if isText {
		handler = slog.NewTextHandler(w, nil)
	} else {
		handler = slog.NewJSONHandler(w, nil)
	}
	return slog.New(handler)
}

// Logf logs informational messages using slog's Info level.
func (l *SlogLogger) Logf(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

// Errorf logs error messages using slog's Error level.
func (l *SlogLogger) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}
