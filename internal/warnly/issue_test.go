package warnly_test

import (
	"testing"

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
