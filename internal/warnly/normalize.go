package warnly

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	whitespacePattern = regexp.MustCompile(`\s+`)

	timestampPatterns = []*regexp.Regexp{
		// ISO 8601 timestamps
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`),
		// RFC3339 timestamps
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`),
		// Unix timestamps
		regexp.MustCompile(`\b\d{10}\b`),
		// Date patterns
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),
		regexp.MustCompile(`\d{2}/\d{2}/\d{4}`),
		regexp.MustCompile(`\d{2}-\d{2}-\d{4}`),
		// Time patterns
		regexp.MustCompile(`\d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`\d{1,2}:\d{2}\s*(?:AM|PM)`),
	}

	// Numeric patterns.
	numericPatterns = []*regexp.Regexp{
		// Large numbers (IDs, timestamps, etc.)
		regexp.MustCompile(`\b\d{6,}\b`),
		// Decimal numbers
		regexp.MustCompile(`\b\d+\.\d+\b`),
		// Hexadecimal values
		regexp.MustCompile(`\b0x[a-fA-F0-9]+\b`),
		// Memory addresses
		regexp.MustCompile(`\b[a-fA-F0-9]{8,}\b`),
	}

	// URL patterns.
	urlPatterns = []*regexp.Regexp{
		// Full URLs
		regexp.MustCompile(`https?://\S+`),
		// File paths (Unix and Windows)
		regexp.MustCompile(`(?:/[^/\s]+)+/[^/\s]*`),
		regexp.MustCompile(`[A-Za-z]:\\(?:[^\\/:*?"<>|\r\n]+\\)*[^\\/:*?"<>|\r\n]*`),
	}

	// UUID patterns.
	uuidPattern = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)

	// IP address patterns.
	ipPatterns = []*regexp.Regexp{
		// IPv4
		regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		// IPv6 (simplified)
		regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`),
	}

	// Email patterns.
	emailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// Generic ID patterns.
	idPatterns = []*regexp.Regexp{
		// Various ID formats
		regexp.MustCompile(`\bid[=:]\s*\d+\b`),
		regexp.MustCompile(`\buserid[=:]\s*\d+\b`),
		regexp.MustCompile(`\buser_id[=:]\s*\d+\b`),
		regexp.MustCompile(`\bsession[=:]\s*[a-zA-Z0-9]+\b`),
		regexp.MustCompile(`\btoken[=:]\s*[a-zA-Z0-9]+\b`),
	}

	// Version patterns.
	versionPattern = regexp.MustCompile(`\bv?\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?\b`)
)

func GetNormalizedHash(event *EventBody) (string, error) {
	normalizedEvent := *event
	normalizedEvent.Exception = NormalizeStackTrace(event.Exception)
	normalizedEvent.Message = NormalizeMessage(event.Message)

	if len(normalizedEvent.Exception) > 0 && len(normalizedEvent.Exception[0].StackTrace.Frames) > 0 {
		return GetHashByNormalizedStackTrace(&normalizedEvent)
	}

	return GetHashByNormalizedMessage(&normalizedEvent)
}

func GetHashByNormalizedStackTrace(event *EventBody) (string, error) {
	h := md5.New() //nolint:gosec // Non-crypto use

	for i := range event.Exception {
		if _, err := h.Write([]byte(event.Exception[i].Type)); err != nil {
			return "", fmt.Errorf("md5: write type: %w", err)
		}

		inAppFrames := []Frame{}
		for j := range event.Exception[i].StackTrace.Frames {
			frame := event.Exception[i].StackTrace.Frames[j]
			if frame.InApp {
				inAppFrames = append(inAppFrames, frame)
			}
		}

		// If no in-app frames, use all frames
		framesToHash := inAppFrames
		if len(inAppFrames) == 0 {
			framesToHash = event.Exception[i].StackTrace.Frames
		}

		for j := range framesToHash {
			frame := framesToHash[j]
			if _, err := h.Write([]byte(frame.Module)); err != nil {
				return "", fmt.Errorf("md5: write module: %w", err)
			}
			if _, err := h.Write([]byte(frame.Function)); err != nil {
				return "", fmt.Errorf("md5: write function: %w", err)
			}
			// Include line number but normalized (rounded to reduce noise)
			normalizedLineNo := (frame.LineNo / 10) * 10 // Round to nearest 10
			if _, err := h.Write(fmt.Appendf(nil, "%d", normalizedLineNo)); err != nil {
				return "", fmt.Errorf("md5: write line number: %w", err)
			}
		}

		normalizedValue := NormalizeMessage(event.Exception[i].Value)
		if _, err := h.Write([]byte(normalizedValue)); err != nil {
			return "", fmt.Errorf("md5: write normalized value: %w", err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetHashByNormalizedMessage(event *EventBody) (string, error) {
	h := md5.New() //nolint:gosec // Non-crypto use

	normalizedMessage := NormalizeMessage(event.Message)
	if _, err := h.Write([]byte(normalizedMessage)); err != nil {
		return "", fmt.Errorf("md5: write normalized message: %w", err)
	}

	if _, err := h.Write([]byte(event.Level)); err != nil {
		return "", fmt.Errorf("md5: write level: %w", err)
	}
	if _, err := h.Write([]byte(event.Platform)); err != nil {
		return "", fmt.Errorf("md5: write platform: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// NormalizeStackTrace normalizes stack trace for consistent grouping.
func NormalizeStackTrace(exceptions []Exception) []Exception {
	if len(exceptions) == 0 {
		return exceptions
	}

	normalized := make([]Exception, len(exceptions))
	copy(normalized, exceptions)

	for i := range normalized {
		normalized[i].Value = NormalizeMessage(normalized[i].Value)

		for j := range normalized[i].StackTrace.Frames {
			frame := &normalized[i].StackTrace.Frames[j]
			frame.ContextLine = NormalizeMessage(frame.ContextLine)

			for k, line := range frame.PreContext {
				frame.PreContext[k] = NormalizeMessage(line)
			}
			for k, line := range frame.PostContext {
				frame.PostContext[k] = NormalizeMessage(line)
			}
		}
	}

	return normalized
}

// NormalizeMessage normalizes an error message for grouping.
func NormalizeMessage(message string) string {
	if message == "" {
		return message
	}

	normalized := message

	for i := range timestampPatterns {
		normalized = timestampPatterns[i].ReplaceAllString(normalized, "<timestamp>")
	}

	normalized = uuidPattern.ReplaceAllString(normalized, "<uuid>")

	for i := range urlPatterns {
		normalized = urlPatterns[i].ReplaceAllString(normalized, "<path>")
	}

	for i := range ipPatterns {
		normalized = ipPatterns[i].ReplaceAllString(normalized, "<ip>")
	}

	normalized = emailPattern.ReplaceAllString(normalized, "<email>")

	normalized = versionPattern.ReplaceAllString(normalized, "<version>")

	for i := range idPatterns {
		normalized = idPatterns[i].ReplaceAllString(normalized, strings.Split(idPatterns[i].String(), `[=:]`)[0]+"=<id>")
	}

	for i := range numericPatterns {
		normalized = numericPatterns[i].ReplaceAllString(normalized, "<number>")
	}

	normalized = whitespacePattern.ReplaceAllString(normalized, " ")

	normalized = strings.TrimSpace(normalized)

	return normalized
}
