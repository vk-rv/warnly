package warnly_test

import (
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

func TestAssignments_AssignedUser(t *testing.T) {
	t.Parallel()

	teammate1 := &warnly.Teammate{
		ID:       1,
		Name:     "John",
		Surname:  "Doe",
		Username: "johndoe",
		Email:    "john@example.com",
	}
	teammate2 := &warnly.Teammate{
		ID:       2,
		Name:     "Jane",
		Surname:  "Smith",
		Username: "janesmith",
		Email:    "jane@example.com",
	}

	//nolint:govet // ignore
	tests := []struct {
		name        string
		assignments warnly.Assignments
		issueID     int64
		wantUser    *warnly.Teammate
		wantOk      bool
	}{
		{
			name: "nil map",
			assignments: warnly.Assignments{
				IssueToAssigned: nil,
			},
			issueID:  1,
			wantUser: nil,
			wantOk:   false,
		},
		{
			name: "issue assigned to teammate1",
			assignments: warnly.Assignments{
				IssueToAssigned: map[int64]*warnly.Teammate{
					1: teammate1,
				},
			},
			issueID:  1,
			wantUser: teammate1,
			wantOk:   true,
		},
		{
			name: "issue assigned to teammate2",
			assignments: warnly.Assignments{
				IssueToAssigned: map[int64]*warnly.Teammate{
					2: teammate2,
				},
			},
			issueID:  2,
			wantUser: teammate2,
			wantOk:   true,
		},
		{
			name: "issue not assigned",
			assignments: warnly.Assignments{
				IssueToAssigned: map[int64]*warnly.Teammate{
					1: teammate1,
				},
			},
			issueID:  2,
			wantUser: nil,
			wantOk:   false,
		},
		{
			name: "empty map",
			assignments: warnly.Assignments{
				IssueToAssigned: map[int64]*warnly.Teammate{},
			},
			issueID:  1,
			wantUser: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotUser, gotOk := tt.assignments.AssignedUser(tt.issueID)
			if gotUser != tt.wantUser || gotOk != tt.wantOk {
				t.Errorf("Assignments.AssignedUser() = (%v, %v), want (%v, %v)", gotUser, gotOk, tt.wantUser, tt.wantOk)
			}
		})
	}
}
