// Package alert provides the implementation of the warnly.AlertService interface.
package alert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertService implements warnly.AlertService interface.
type AlertService struct {
	now          func() time.Time
	alertStore   warnly.AlertStore
	projectStore warnly.ProjectStore
	teamStore    warnly.TeamStore
	logger       *slog.Logger
}

// NewAlertService is a constructor of AlertService.
func NewAlertService(
	alertStore warnly.AlertStore,
	projectStore warnly.ProjectStore,
	teamStore warnly.TeamStore,
	now func() time.Time,
	logger *slog.Logger,
) *AlertService {
	return &AlertService{
		alertStore:   alertStore,
		projectStore: projectStore,
		teamStore:    teamStore,
		now:          now,
		logger:       logger,
	}
}

// ListAlerts returns a list of alerts for the given criteria.
func (s *AlertService) ListAlerts(ctx context.Context, req *warnly.ListAlertsRequest) (*warnly.ListAlertsResult, error) {
	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return nil, err
	}

	teamIDs := make([]int, len(teams))
	for i := range teams {
		teamIDs[i] = teams[i].ID
	}

	if req.TeamName != "" {
		foundTeam := false
		for i := range teams {
			if teams[i].Name == req.TeamName {
				teamIDs = []int{teams[i].ID}
				foundTeam = true
				break
			}
		}
		if !foundTeam {
			teamIDs = []int{}
		}
	}

	alerts, totalCount, err := s.alertStore.ListAlerts(ctx, teamIDs, req.ProjectName, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}

	projects, err := s.projectStore.ListProjects(ctx, teamIDs, "")
	if err != nil {
		return nil, err
	}

	return &warnly.ListAlertsResult{
		Request:     req,
		Alerts:      alerts,
		Projects:    projects,
		Teams:       teams,
		TotalAlerts: totalCount,
	}, nil
}

// CreateAlert creates a new alert.
func (s *AlertService) CreateAlert(ctx context.Context, req *warnly.CreateAlertRequest) (*warnly.Alert, error) {
	project, err := s.projectStore.GetProject(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}

	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return nil, err
	}

	hasAccess := false
	for i := range teams {
		if teams[i].ID == project.TeamID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return nil, errors.New("user does not have access to this project")
	}

	description := fmt.Sprintf("Alert when more than %d %s in %s",
		req.Threshold,
		getConditionText(req.Condition),
		getTimeframeText(req.Timeframe),
	)

	now := s.now().UTC()
	alert := &warnly.Alert{
		CreatedAt:    now,
		UpdatedAt:    now,
		RuleName:     req.RuleName,
		Description:  description,
		Status:       warnly.AlertStatusActive,
		ProjectID:    req.ProjectID,
		TeamID:       project.TeamID,
		Threshold:    req.Threshold,
		Condition:    req.Condition,
		Timeframe:    req.Timeframe,
		HighPriority: req.HighPriority,
	}

	if err := s.alertStore.CreateAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// UpdateAlert updates an existing alert.
func (s *AlertService) UpdateAlert(ctx context.Context, req *warnly.UpdateAlertRequest) (*warnly.Alert, error) {
	alert, err := s.alertStore.GetAlert(ctx, req.AlertID)
	if err != nil {
		return nil, err
	}

	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return nil, err
	}

	hasAccess := false
	for i := range teams {
		if teams[i].ID == alert.TeamID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		return nil, errors.New("user does not have access to this alert")
	}

	alert.UpdatedAt = s.now().UTC()
	alert.RuleName = req.RuleName
	alert.Threshold = req.Threshold
	alert.Condition = req.Condition
	alert.Timeframe = req.Timeframe
	alert.HighPriority = req.HighPriority
	if req.Status != "" {
		alert.Status = req.Status
	}

	alert.Description = fmt.Sprintf("Alert when more than %d %s in %s",
		req.Threshold,
		getConditionText(req.Condition),
		getTimeframeText(req.Timeframe),
	)

	if err := s.alertStore.UpdateAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// DeleteAlert deletes an alert by ID.
func (s *AlertService) DeleteAlert(ctx context.Context, alertID int, user *warnly.User) error {
	alert, err := s.alertStore.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return err
	}

	hasAccess := false
	for i := range teams {
		if teams[i].ID == alert.TeamID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return errors.New("user does not have access to this alert")
	}

	if err := s.alertStore.DeleteAlert(ctx, alertID); err != nil {
		return err
	}

	return nil
}

// GetAlert returns an alert by ID.
func (s *AlertService) GetAlert(ctx context.Context, alertID int, user *warnly.User) (*warnly.Alert, error) {
	alert, err := s.alertStore.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}

	teams, err := s.teamStore.ListTeams(ctx, int(user.ID))
	if err != nil {
		return nil, err
	}

	hasAccess := false
	for i := range teams {
		if teams[i].ID == alert.TeamID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return nil, errors.New("user does not have access to this alert")
	}

	return alert, nil
}

func getConditionText(condition warnly.AlertCondition) string {
	switch condition {
	case warnly.AlertConditionOccurrences:
		return "occurrences"
	case warnly.AlertConditionUsers:
		return "users affected"
	default:
		return "occurrences"
	}
}

func getTimeframeText(timeframe warnly.AlertTimeframe) string {
	switch timeframe {
	case warnly.AlertTimeframe1Min:
		return "1 minute"
	case warnly.AlertTimeframe5Min:
		return "5 minutes"
	case warnly.AlertTimeframe15Min:
		return "15 minutes"
	case warnly.AlertTimeframe1Hour:
		return "1 hour"
	case warnly.AlertTimeframe1Day:
		return "1 day"
	case warnly.AlertTimeframe1Week:
		return "1 week"
	case warnly.AlertTimeframe30Days:
		return "30 days"
	default:
		return "1 hour"
	}
}
