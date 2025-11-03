package warnly

import (
	"context"
	"time"
)

// AlertService encapsulates service domain logic.
type AlertService interface {
	// ListAlerts returns a list of alerts for the given criteria.
	ListAlerts(ctx context.Context, req *ListAlertsRequest) (*ListAlertsResult, error)
	// CreateAlert creates a new alert.
	CreateAlert(ctx context.Context, req *CreateAlertRequest) (*Alert, error)
	// UpdateAlert updates an existing alert.
	UpdateAlert(ctx context.Context, req *UpdateAlertRequest) (*Alert, error)
	// DeleteAlert deletes an alert by ID.
	DeleteAlert(ctx context.Context, alertID int, user *User) error
	// GetAlert returns an alert by ID.
	GetAlert(ctx context.Context, alertID int, user *User) (*Alert, error)
}

// AlertStore encapsulates the alert storage.
type AlertStore interface {
	// ListAlerts returns a list of alerts for the given criteria.
	ListAlerts(ctx context.Context, teamIDs []int, projectName string, offset, limit int) ([]Alert, int, error)
	// CreateAlert creates a new alert.
	CreateAlert(ctx context.Context, alert *Alert) error
	// UpdateAlert updates an existing alert.
	UpdateAlert(ctx context.Context, alert *Alert) error
	// DeleteAlert deletes an alert by ID.
	DeleteAlert(ctx context.Context, alertID int) error
	// GetAlert returns an alert by ID.
	GetAlert(ctx context.Context, alertID int) (*Alert, error)
	// ListAlertsByProject returns alerts for a project.
	ListAlertsByProject(ctx context.Context, projectID int) ([]Alert, error)
}

type AlertStatus string

const (
	// AlertStatusActive represents an active alert.
	AlertStatusActive AlertStatus = "Active"
	// AlertStatusInactive represents an inactive alert.
	AlertStatusInactive AlertStatus = "Inactive"
	// AlertStatusTriggered represents a triggered alert.
	AlertStatusTriggered AlertStatus = "Triggered"
)

type AlertCondition int

const (
	// AlertConditionOccurrences - when threshold number of occurrences is reached.
	AlertConditionOccurrences AlertCondition = 1
	// AlertConditionUsers - when threshold number of users is affected.
	AlertConditionUsers AlertCondition = 2
)

type AlertTimeframe int

const (
	// AlertTimeframe1Min - 1 minute.
	AlertTimeframe1Min AlertTimeframe = 1
	// AlertTimeframe5Min - 5 minutes.
	AlertTimeframe5Min AlertTimeframe = 2
	// AlertTimeframe15Min - 15 minutes.
	AlertTimeframe15Min AlertTimeframe = 3
	// AlertTimeframe1Hour - 1 hour.
	AlertTimeframe1Hour AlertTimeframe = 4
	// AlertTimeframe1Day - 1 day.
	AlertTimeframe1Day AlertTimeframe = 5
	// AlertTimeframe1Week - 1 week.
	AlertTimeframe1Week AlertTimeframe = 6
	// AlertTimeframe30Days - 30 days.
	AlertTimeframe30Days AlertTimeframe = 7
)

// Alert represents an alert rule.
type Alert struct {
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastTriggeredAt    *time.Time
	ResolvedAt         *time.Time
	NotificationSentAt *time.Time
	RuleName           string
	Description        string
	Status             AlertStatus
	ID                 int
	ProjectID          int
	TeamID             int
	Threshold          int
	Condition          AlertCondition // 1 = occurrences, 2 = users affected
	Timeframe          AlertTimeframe // 1=1min, 2=5min, 3=15min, 4=1h, 5=1d, 6=1w, 7=30d
	HighPriority       bool
}

// GetTimeframeDuration returns the duration for the timeframe.
func (a *Alert) GetTimeframeDuration() time.Duration {
	switch a.Timeframe {
	case AlertTimeframe1Min:
		return time.Minute
	case AlertTimeframe5Min:
		return 5 * time.Minute
	case AlertTimeframe15Min:
		return 15 * time.Minute
	case AlertTimeframe1Hour:
		return time.Hour
	case AlertTimeframe1Day:
		return 24 * time.Hour
	case AlertTimeframe1Week:
		return 7 * 24 * time.Hour
	case AlertTimeframe30Days:
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// ListAlertsRequest is used to specify criteria for listing alerts.
type ListAlertsRequest struct {
	User        *User
	TeamName    string
	ProjectName string
	Offset      int
	Limit       int
}

// ListAlertsResult contains the result of listing alerts.
type ListAlertsResult struct {
	Request     *ListAlertsRequest
	Alerts      []Alert
	Projects    []Project
	Teams       []Team
	TotalAlerts int
}

// CreateAlertRequest is a request to create a new alert.
type CreateAlertRequest struct {
	User         *User
	RuleName     string
	ProjectID    int
	Threshold    int
	Condition    AlertCondition
	Timeframe    AlertTimeframe
	HighPriority bool
}

// UpdateAlertRequest is a request to update an alert.
type UpdateAlertRequest struct {
	User         *User
	RuleName     string
	Status       AlertStatus
	AlertID      int
	Threshold    int
	Condition    AlertCondition
	Timeframe    AlertTimeframe
	HighPriority bool
}
