package stdlog_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/vk-rv/warnly/internal/stdlog"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slogLogger := slog.New(handler)

	logger := stdlog.NewLogger(slogLogger)
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestSlogLogger_Logf(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slogLogger := slog.New(handler)

	logger := stdlog.NewLogger(slogLogger)

	logger.Logf("test message %s", "arg")

	output := buf.String()
	if !strings.Contains(output, "test message arg") {
		t.Errorf("Logf() output does not contain expected message: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Logf() output does not contain INFO level: %s", output)
	}
}

func TestSlogLogger_Errorf(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slogLogger := slog.New(handler)

	logger := stdlog.NewLogger(slogLogger)

	logger.Errorf("error message %d", 123)

	output := buf.String()
	if !strings.Contains(output, "error message 123") {
		t.Errorf("Errorf() output does not contain expected message: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Errorf() output does not contain ERROR level: %s", output)
	}
}

func TestNewSlogLogger_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	slogLogger := slog.New(handler)

	logger := stdlog.NewLogger(slogLogger)

	logger.Logf("test message")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("Logf() JSON output does not contain expected message: %s", output)
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("Logf() JSON output does not contain INFO level: %s", output)
	}
}
