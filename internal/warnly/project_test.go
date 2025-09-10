package warnly_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"", 0, false},
		{"2h", 2 * time.Hour, false},
		{"5m", 5 * time.Minute, false},
		{"10s", 10 * time.Second, false},
		{"3d", 3 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"100h", 100 * time.Hour, false},
		{"0h", 0, false},
		{"1x", 0, true},
		{"abc", 0, true},
		{"5", 0, true},
		{"-2h", 2 * time.Hour, false},
	}

	for _, tt := range tests {
		got, err := warnly.ParseDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Fatalf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseTimeRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantStart time.Time
		wantEnd   time.Time
		start     string
		end       string
		wantErr   bool
	}{
		{
			start:     "2025-06-20T00:00:00",
			end:       "2025-06-26T23:59:59",
			wantStart: time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2025, 6, 26, 23, 59, 59, 0, time.UTC),
			wantErr:   false,
		},
		{
			start:     "2024-01-01T12:00:00",
			end:       "2024-01-02T12:00:00",
			wantStart: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			start:   "invalid",
			end:     "2025-06-26T23:59:59",
			wantErr: true,
		},
		{
			start:   "2025-06-20T00:00:00",
			end:     "invalid",
			wantErr: true,
		},
		{
			start:   "",
			end:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		gotStart, gotEnd, err := warnly.ParseTimeRange(tt.start, tt.end)
		if tt.wantErr {
			require.Error(t, err, "ParseTimeRange(%q, %q) expected error", tt.start, tt.end)
			continue
		}
		require.NoError(t, err, "ParseTimeRange(%q, %q) unexpected error", tt.start, tt.end)
		require.True(t, gotStart.Equal(tt.wantStart), "ParseTimeRange(%q, %q) gotStart = %v, want %v", tt.start, tt.end, gotStart, tt.wantStart)
		require.True(t, gotEnd.Equal(tt.wantEnd), "ParseTimeRange(%q, %q) gotEnd = %v, want %v", tt.start, tt.end, gotEnd, tt.wantEnd)
	}
}
