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

func TestParseQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected []warnly.QueryToken
	}{
		{
			name:     "empty query",
			query:    "",
			expected: []warnly.QueryToken{},
		},
		{
			name:  "single tag",
			query: "release:nordland@0.1.0",
			expected: []warnly.QueryToken{
				{Key: "release", Operator: "is", Value: "nordland@0.1.0"},
			},
		},
		{
			name:  "tag with not",
			query: "server_name:!Olegs-MacBook-Pro.local",
			expected: []warnly.QueryToken{
				{Key: "server_name", Operator: "is not", Value: "Olegs-MacBook-Pro.local"},
			},
		},
		{
			name:  "tag with quoted value",
			query: `server_name:"delein computer"`,
			expected: []warnly.QueryToken{
				{Key: "server_name", Operator: "is", Value: "delein computer"},
			},
		},
		{
			name:  "raw text",
			query: `"pro error"`,
			expected: []warnly.QueryToken{
				{Value: "pro error", IsRawText: true},
			},
		},
		{
			name:  "multiple tokens",
			query: `release:nordland@0.1.0 level:error server_name:!Olegs-MacBook-Pro.local "pro error" "div error" server_name:"delein computer"`,
			expected: []warnly.QueryToken{
				{Key: "release", Operator: "is", Value: "nordland@0.1.0"},
				{Key: "level", Operator: "is", Value: "error"},
				{Key: "server_name", Operator: "is not", Value: "Olegs-MacBook-Pro.local"},
				{Value: "pro error", IsRawText: true},
				{Value: "div error", IsRawText: true},
				{Key: "server_name", Operator: "is", Value: "delein computer"},
			},
		},
		{
			name:  "single quotes",
			query: "tag:'value with spaces'",
			expected: []warnly.QueryToken{
				{Key: "tag", Operator: "is", Value: "value with spaces"},
			},
		},
		{
			name:  "mixed quotes",
			query: `"text" key:value`,
			expected: []warnly.QueryToken{
				{Value: "text", IsRawText: true},
				{Key: "key", Operator: "is", Value: "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.ParseQuery(tt.query)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(result))
				return
			}
			for i, token := range result {
				expected := tt.expected[i]
				if token != expected {
					t.Errorf("token %d: expected %+v, got %+v", i, expected, token)
				}
			}
		})
	}
}

func TestTeammateAvatarInitials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		teammate *warnly.Teammate
		want     string
	}{
		{
			name: "normal case",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "Doe",
			},
			want: "JD",
		},
		{
			name: "single character names",
			teammate: &warnly.Teammate{
				Name:    "A",
				Surname: "B",
			},
			want: "AB",
		},
		{
			name: "long names",
			teammate: &warnly.Teammate{
				Name:    "Alexander",
				Surname: "Sokolov",
			},
			want: "AS",
		},
		{
			name: "lowercase names",
			teammate: &warnly.Teammate{
				Name:    "john",
				Surname: "doe",
			},
			want: "JD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.teammate.AvatarInitials()
			if result != tt.want {
				t.Errorf("AvatarInitials() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestListIssuesResultNoIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result   *warnly.ListIssuesResult
		name     string
		wantTrue bool
	}{
		{
			name:     "empty issues",
			result:   &warnly.ListIssuesResult{Issues: []warnly.IssueEntry{}},
			wantTrue: true,
		},
		{
			name:     "nil issues",
			result:   &warnly.ListIssuesResult{Issues: nil},
			wantTrue: true,
		},
		{
			name: "single issue",
			result: &warnly.ListIssuesResult{
				Issues: []warnly.IssueEntry{
					{ID: 1, Message: "Error 1"},
				},
			},
			wantTrue: false,
		},
		{
			name: "multiple issues",
			result: &warnly.ListIssuesResult{
				Issues: []warnly.IssueEntry{
					{ID: 1, Message: "Error 1"},
					{ID: 2, Message: "Error 2"},
					{ID: 3, Message: "Error 3"},
				},
			},
			wantTrue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.result.NoIssues()
			if result != tt.wantTrue {
				t.Errorf("NoIssues() = %v, want %v", result, tt.wantTrue)
			}
		})
	}
}

func TestTeammateFullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		teammate *warnly.Teammate
		want     string
	}{
		{
			name: "normal case",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "Doe",
			},
			want: "John Doe",
		},
		{
			name: "single character names",
			teammate: &warnly.Teammate{
				Name:    "A",
				Surname: "B",
			},
			want: "A B",
		},
		{
			name: "long names",
			teammate: &warnly.Teammate{
				Name:    "Alexander",
				Surname: "Sokolov",
			},
			want: "Alexander Sokolov",
		},
		{
			name: "mixed case",
			teammate: &warnly.Teammate{
				Name:    "john",
				Surname: "DOE",
			},
			want: "john DOE",
		},
		{
			name: "empty name",
			teammate: &warnly.Teammate{
				Name:    "",
				Surname: "Doe",
			},
			want: " Doe",
		},
		{
			name: "empty surname",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "",
			},
			want: "John ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.teammate.FullName()
			if result != tt.want {
				t.Errorf("FullName() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	mockNow := func() time.Time { return now }

	tests := []struct {
		t          time.Time
		name       string
		wantWide   string
		wantNarrow string
		narrow     bool
	}{
		{
			name:       "30 seconds ago",
			t:          now.Add(-30 * time.Second),
			narrow:     false,
			wantWide:   "30 seconds",
			wantNarrow: "30sec",
		},
		{
			name:       "1 second ago",
			t:          now.Add(-time.Second),
			narrow:     false,
			wantWide:   "1 second",
			wantNarrow: "1sec",
		},
		{
			name:       "5 minutes ago",
			t:          now.Add(-5 * time.Minute),
			narrow:     false,
			wantWide:   "5 minutes",
			wantNarrow: "5min",
		},
		{
			name:       "2 hours ago",
			t:          now.Add(-2 * time.Hour),
			narrow:     false,
			wantWide:   "2 hours",
			wantNarrow: "2h",
		},
		{
			name:       "10 days ago",
			t:          now.Add(-10 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "10 days",
			wantNarrow: "10d",
		},
		{
			name:       "2 months ago",
			t:          now.Add(-60 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "2 months",
			wantNarrow: "2mo",
		},
		{
			name:       "2 years ago",
			t:          now.Add(-730 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "2 years",
			wantNarrow: "2y",
		},
		{
			name:       "30 seconds ago (narrow)",
			t:          now.Add(-30 * time.Second),
			narrow:     true,
			wantWide:   "",
			wantNarrow: "30sec",
		},
		{
			name:       "5 minutes ago (narrow)",
			t:          now.Add(-5 * time.Minute),
			narrow:     true,
			wantWide:   "",
			wantNarrow: "5min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.TimeAgo(mockNow, tt.t, tt.narrow)
			want := tt.wantWide
			if tt.narrow {
				want = tt.wantNarrow
			}
			if result != want {
				t.Errorf("TimeAgo() = %q, want %q", result, want)
			}
		})
	}
}
