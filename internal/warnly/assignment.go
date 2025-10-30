package warnly

import (
	"context"
	"database/sql"
	"time"
)

// Assignment represents the assignment of an issue to a user.
type Assignment struct {
	AssignedAt       time.Time `json:"assigned_at"`
	IssueID          int64     `json:"issue_id"`
	AssignedToUserID int64     `json:"assigned_to_user_id"`
	AssignedToTeamID int64     `json:"assigned_to_team_id"`
	AssignedByUserID int64     `json:"assigned_by_user_id"`
}

// AssignedUser represents the result of assigned user to an issue.
type AssignedUser struct {
	IssueID          int64
	AssignedToUserID sql.NullInt64
}

type Assignments struct {
	IssueToAssigned map[int64]*Teammate
}

// AssignedUser returns the assigned user for the given issue.
func (a *Assignments) AssignedUser(issueID int64) (*Teammate, bool) {
	if a.IssueToAssigned == nil {
		return nil, false
	}
	assigned, ok := a.IssueToAssigned[issueID]
	return assigned, ok
}

// AssingmentStore is the interface for the assignment storage.
type AssingmentStore interface {
	// CreateAssingment creates a new issue assignment in the database.
	CreateAssingment(ctx context.Context, assignment *Assignment) error
	// DeleteAssignment unassigns an issue from a user.
	DeleteAssignment(ctx context.Context, issueID int64) error
	// ListAssingments lists all assignments for a given issue.
	ListAssingments(ctx context.Context, issueIDs []int64) ([]*AssignedUser, error)
	// ListAssignedFilters gets filters for assigned issues.
	ListAssignedFilters(ctx context.Context, criteria *GetAssignedFiltersCriteria) ([]Filter, error)
}

// AssignIssueRequest represents the request to assign an issue to a user.
type AssignIssueRequest struct {
	User      *User `json:"user"`
	IssueID   int   `json:"issue_id"`
	ProjectID int   `json:"project_id"`
	UserID    int   `json:"user_id"`
}

// UnassignIssueRequest represents the request to unassign an issue from a user.
type UnassignIssueRequest struct {
	User      *User `json:"user"`
	IssueID   int   `json:"issue_id"`
	ProjectID int   `json:"project_id"`
}
