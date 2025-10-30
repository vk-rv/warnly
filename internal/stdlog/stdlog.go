// Package stdlog provides a logger implementation using the standard library's slog package
// used as adapter for logging interfaces.
package stdlog

import (
	"fmt"
	"log/slog"
	"os"
)

// LogOutput represents the output destination for the logger.
type LogOutput = string

const (
	// Stdout is the standard output stream.
	Stdout LogOutput = "stdout"
	// Stderr is the standard error stream.
	Stderr LogOutput = "stderr"
)

// SlogLogger is the implementation of Logger using slog.
type SlogLogger struct {
	logger *slog.Logger
}

// NewLogger creates a new SlogLogger instance.
func NewLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{logger: logger}
}

// NewSlogLogger creates a new slog.Logger instance with the specified destination output and format.
func NewSlogLogger(output LogOutput, isText bool) *slog.Logger {
	var (
		out     *os.File
		handler slog.Handler
	)

	if output == Stdout {
		out = os.Stdout
	} else {
		out = os.Stderr
	}

	if isText {
		handler = slog.NewTextHandler(out, nil)
	} else {
		handler = slog.NewJSONHandler(out, nil)
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
