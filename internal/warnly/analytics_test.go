package warnly_test

import (
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

func TestEventEntry_DisplayUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry warnly.EventEntry
		want  string
	}{
		{
			name:  "with email",
			entry: warnly.EventEntry{UserEmail: "test@example.com"},
			want:  "test@example.com",
		},
		{
			name:  "with username",
			entry: warnly.EventEntry{UserUsername: "testuser"},
			want:  "testuser",
		},
		{
			name:  "with user id",
			entry: warnly.EventEntry{User: "123"},
			want:  "123",
		},
		{
			name:  "with name",
			entry: warnly.EventEntry{UserName: "Test User"},
			want:  "Test User",
		},
		{
			name:  "no value",
			entry: warnly.EventEntry{},
			want:  "(no value)",
		},
		{
			name:  "email takes precedence over username",
			entry: warnly.EventEntry{UserEmail: "test@example.com", UserUsername: "testuser"},
			want:  "test@example.com",
		},
		{
			name:  "username takes precedence over user id",
			entry: warnly.EventEntry{UserUsername: "testuser", User: "123"},
			want:  "testuser",
		},
		{
			name:  "user id takes precedence over name",
			entry: warnly.EventEntry{User: "123", UserName: "Test User"},
			want:  "123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.entry.DisplayUser(); got != tt.want {
				t.Errorf("EventEntry.DisplayUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
