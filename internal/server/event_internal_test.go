package server

import (
	"testing"
)

func TestProjectKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		wantKey string
		wantErr bool
	}{
		{
			name:    "happy path",
			header:  "sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=urzovxt",
			wantKey: "urzovxt",
			wantErr: false,
		},
		{
			name:    "missing sentry_key",
			header:  "sentry_version=7, sentry_client=sentry.go/0.30.0",
			wantKey: "",
			wantErr: true,
		},
		{
			name:    "empty header",
			header:  "",
			wantKey: "",
			wantErr: true,
		},
		{
			name:    "malformed header",
			header:  "sentry_version=7 sentry_client=sentry.go/0.30.0 sentry_key=urzovxt",
			wantKey: "",
			wantErr: true,
		},
		{
			name:    "sentry_key at start",
			header:  "sentry_key=urzovxt, sentry_version=7, sentry_client=sentry.go/0.30.0",
			wantKey: "urzovxt",
			wantErr: false,
		},
		{
			name:    "sentry_key at end",
			header:  "sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=urzovxt",
			wantKey: "urzovxt",
			wantErr: false,
		},
		{
			name:    "multiple keys",
			header:  "sentry_version=7, sentry_key=abc123, sentry_client=sentry.go/0.30.0, sentry_key=urzovxt",
			wantKey: "abc123", // assuming first occurrence is used
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotKey, err := projectKey(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("projectKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotKey != tt.wantKey {
				t.Errorf("projectKey() = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}
