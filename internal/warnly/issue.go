package warnly

import (
	"context"
	"time"
)

// Issue represents a collection of error events mapped by their hash.
type Issue struct {
	FirstSeen   time.Time     `json:"first_seen"`
	LastSeen    time.Time     `json:"last_seen"`
	Hash        string        `json:"hash"`
	Message     string        `json:"message"`
	View        string        `json:"view"`
	ErrorType   string        `json:"error_type"`
	UUID        UUID          `json:"uuid"`
	ID          int64         `json:"id"`
	NumComments int           `json:"num_comments"`
	ProjectID   int           `json:"project_id"`
	Priority    IssuePriority `json:"priority"`
}

// IssueMetrics represents the metrics of an issue.
type IssueMetrics struct {
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	GID       uint64    `json:"gid"`
	TimesSeen uint64    `json:"times_seen"`
	UserCount uint64    `json:"user_count"`
}

func GetMetrics(metrics []IssueMetrics, id int64) (IssueMetrics, bool) {
	for i := range metrics {
		if int64(metrics[i].GID) == id {
			return metrics[i], true
		}
	}
	return IssueMetrics{}, false
}

type IssueInfo struct {
	UUID string `json:"uuid"`
	Hash string `json:"hash"`
	ID   int64  `json:"id"`
}

type IssuePriority int

const (
	PriorityLow IssuePriority = iota + 1
	PriorityMedium
	PriorityHigh
)

var AllowedPriorities = [...]IssuePriority{
	PriorityLow,
	PriorityMedium,
	PriorityHigh,
}

func (p IssuePriority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Med"
	case PriorityHigh:
		return "High"
	default:
		return "Unknown"
	}
}

type IssueStore interface {
	// GetIssue returns an issue by hash.
	GetIssue(ctx context.Context, criteria GetIssueCriteria) (*Issue, error)
	// GetIssueByID returns an issue by ID.
	GetIssueByID(ctx context.Context, id int64) (*Issue, error)
	// StoreIssue stores a new issue.
	StoreIssue(ctx context.Context, issue *Issue) error
	// ListIssues returns a list of issues.
	ListIssues(ctx context.Context, criteria ListIssuesCriteria) ([]Issue, error)
	// UpdateLastSeen updates the last seen time of an issue.
	UpdateLastSeen(ctx context.Context, upd *UpdateLastSeen) error
}

type UpdateLastSeen struct {
	LastSeen  time.Time
	Message   string
	ErrorType string
	View      string
	IssueID   int64
}

// GetIssueCriteria is used to specify criteria for fetching an issue.
type GetIssueCriteria struct {
	// required.
	Hash string
	// required.
	ProjectID int
}

// ListIssuesCriteria is used to specify criteria for listing issues.
type ListIssuesCriteria struct {
	From       time.Time
	To         time.Time
	ProjectIDs []int
}

// IsAllowedIssueType checks whether provided issueType argument is included into predefined
// issue types allowed list.
func IsAllowedIssueType(issueType IssuesType) bool {
	for i := range AllowedIssuesTypes {
		if issueType == AllowedIssuesTypes[i] {
			return true
		}
	}
	return false
}
