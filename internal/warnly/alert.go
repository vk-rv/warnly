package warnly

import "context"

// AlertService encapsulates service domain logic.
type AlertService interface {
	// ListAlerts returns a list of alerts for the given criteria.
	ListAlerts(ctx context.Context, req *ListAlertsRequest) (*ListAlertsResult, error)
}

// AlertStore encapsulates the alert storage.
type AlertStore interface {
	// ListAlerts returns a list of alerts for the given criteria.
	ListAlerts(ctx context.Context, req *ListAlertsRequest) ([]Alert, error)
}

// Alert represents an alert rule.
type Alert struct {
	RuleName    string `json:"rule_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ID          int    `json:"id"`
	ProjectID   int    `json:"project_id"`
	TeamID      int    `json:"team_id"`
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
	RequestedTeam    string
	RequestedProject string
	Request          *ListAlertsRequest
	Alerts           []Alert
	Projects         []Project
	Teams            []Team
	TotalAlerts      int
}
