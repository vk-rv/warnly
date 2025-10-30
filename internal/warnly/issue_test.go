package warnly_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestIsAllowedIssueType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   warnly.IssuesType
		allowed bool
	}{
		{"all issues", "all", true},
		{"new issues", "new", true},
		{"not_allowed", "not_allowed", false},
		{"empty issues", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.IsAllowedIssueType(tt.input)
			require.Equal(t, tt.allowed, result)
		})
	}
}

func TestGetMetrics(t *testing.T) {
	t.Parallel()

	now := time.Now()
	metrics := []warnly.IssueMetrics{
		{GID: 1, TimesSeen: 10, UserCount: 5, FirstSeen: now, LastSeen: now},
		{GID: 2, TimesSeen: 20, UserCount: 10, FirstSeen: now, LastSeen: now},
	}

	//nolint:govet // ignore
	tests := []struct {
		expected warnly.IssueMetrics
		id       int64
		name     string
		found    bool
	}{
		{
			name:     "existing id",
			id:       1,
			expected: metrics[0],
			found:    true,
		},
		{
			name:     "another existing id",
			id:       2,
			expected: metrics[1],
			found:    true,
		},
		{
			name:     "non-existing id",
			id:       3,
			expected: warnly.IssueMetrics{},
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, found := warnly.GetMetrics(metrics, tt.id)
			require.Equal(t, tt.found, found)
			if found {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIssuePriority_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		priority warnly.IssuePriority
	}{
		{"low", "Low", warnly.PriorityLow},
		{"medium", "Med", warnly.PriorityMedium},
		{"high", "High", warnly.PriorityHigh},
		{"unknown", "Unknown", warnly.IssuePriority(999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.priority.String()
			require.Equal(t, tt.expected, result)
		})
	}
}
