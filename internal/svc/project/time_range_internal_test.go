package project

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestProjectTimeRange(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 6, 18, 50, 0, 0, time.UTC)
	svc := &ProjectService{now: func() time.Time { return now }}
	tests := []struct {
		name    string
		req     warnly.ProjectDetailsRequest
		from    time.Time
		to      time.Time
		wantErr bool
	}{
		{name: "direct project link defaults to 24 hours", from: now.Add(-24 * time.Hour), to: now},
		{name: "explicit period", req: warnly.ProjectDetailsRequest{Period: "1h"}, from: now.Add(-time.Hour), to: now},
		{
			name: "custom range",
			req:  warnly.ProjectDetailsRequest{Start: "2026-10-01T12:00:00", End: "2026-10-02T12:00:00"},
			from: time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC),
			to:   time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC),
		},
		{name: "missing end is not a default range", req: warnly.ProjectDetailsRequest{Start: "2026-10-01T12:00:00"}, wantErr: true},
		{name: "missing start is not a default range", req: warnly.ProjectDetailsRequest{End: "2026-10-02T12:00:00"}, wantErr: true},
		{name: "invalid period", req: warnly.ProjectDetailsRequest{Period: "invalid"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			from, to, err := svc.getTimeRange(&tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.from, from)
			require.Equal(t, tt.to, to)
		})
	}
}
