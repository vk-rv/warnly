// Package alert provides the implementation of the warnly.AlertService interface.
package alert

import (
	"context"
	"log/slog"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertService implements warnly.AlertService interface.
type AlertService struct {
	alertStore warnly.AlertStore
	logger     *slog.Logger
}

// NewAlertService is a constructor of AlertService.
func NewAlertService(alertStore warnly.AlertStore, logger *slog.Logger) *AlertService {
	return &AlertService{
		alertStore: alertStore,
		logger:     logger,
	}
}

// ListAlerts returns a list of alerts for the given criteria.
func (s *AlertService) ListAlerts(ctx context.Context, req *warnly.ListAlertsRequest) (*warnly.ListAlertsResult, error) {
	return &warnly.ListAlertsResult{
		RequestedTeam:    req.TeamName,
		RequestedProject: req.ProjectName,
		Request: &warnly.ListAlertsRequest{
			TeamName:    req.TeamName,
			ProjectName: req.ProjectName,
			Offset:      req.Offset,
		},
		Alerts: []warnly.Alert{
			{ID: 1, RuleName: "High Error Rate", Description: "Error rate exceeds 10%", Status: "Active", ProjectID: 1, TeamID: 1},
			{ID: 2, RuleName: "Slow Response Time", Description: "Response time > 5s", Status: "Inactive", ProjectID: 2, TeamID: 2},
			{ID: 3, RuleName: "Memory Usage Spike", Description: "Memory usage > 80%", Status: "Triggered", ProjectID: 1, TeamID: 1},
		},
		Projects: []warnly.Project{
			{ID: 1, Name: "Project A"},
			{ID: 2, Name: "Project B"},
		},
		Teams: []warnly.Team{
			{ID: 1, Name: "Team Alpha"},
			{ID: 2, Name: "Team Beta"},
		},
		TotalAlerts: 3,
	}, nil
}
