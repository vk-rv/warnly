// Package worker provides background workers for processing alerts.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertWorker processes alert rules in the background.
//
//nolint:govet // dont care about it for now
type AlertWorker struct {
	alertStore     warnly.AlertStore
	analyticsStore warnly.AnalyticsStore
	logger         *slog.Logger
	stopCh         chan struct{}
	now            func() time.Time
	interval       time.Duration
	instanceID     string
	mu             sync.Mutex
	running        bool
}

// NewAlertWorker creates a new alert worker.
func NewAlertWorker(
	alertStore warnly.AlertStore,
	analyticsStore warnly.AnalyticsStore,
	now func() time.Time,
	interval time.Duration,
	instanceID string,
	logger *slog.Logger,
) *AlertWorker {
	return &AlertWorker{
		now:            now,
		alertStore:     alertStore,
		analyticsStore: analyticsStore,
		logger:         logger,
		interval:       interval,
		instanceID:     instanceID,
		stopCh:         make(chan struct{}),
	}
}

// Start begins processing alerts in the background.
func (w *AlertWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.logger.Info("alert worker started", slog.String("instance_id", w.instanceID))

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.processAlerts(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("alert worker stopped due to context cancellation")
			return
		case <-w.stopCh:
			w.logger.Info("alert worker stopped")
			return
		case <-ticker.C:
			w.processAlerts(ctx)
		}
	}
}

// Stop stops the alert worker.
func (w *AlertWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopCh)
	w.running = false
}

// processAlerts checks all active alerts and triggers them if needed.
func (w *AlertWorker) processAlerts(ctx context.Context) {
	alerts, _, err := w.alertStore.ListAlerts(ctx, []int{}, "", 0, 1000)
	if err != nil {
		w.logger.Error("failed to list alerts", slog.Any("error", err))
		return
	}

	activeAlerts := make([]warnly.Alert, 0)
	for i := range alerts {
		if alerts[i].Status == warnly.AlertStatusActive {
			activeAlerts = append(activeAlerts, alerts[i])
		}
	}

	if len(activeAlerts) == 0 {
		return
	}

	w.logger.Debug("processing alerts", slog.Int("count", len(activeAlerts)))

	for i := range activeAlerts {
		if err := w.checkAlert(ctx, &activeAlerts[i]); err != nil {
			w.logger.Error("failed to check alert",
				slog.Int("alert_id", activeAlerts[i].ID),
				slog.String("alert_name", activeAlerts[i].RuleName),
				slog.Any("error", err),
			)
		}
	}
}

// checkAlert checks if an alert should be triggered.
func (w *AlertWorker) checkAlert(ctx context.Context, alert *warnly.Alert) error {
	now := w.now().UTC()
	timeframe := alert.GetTimeframeDuration()
	from := now.Add(-timeframe)

	criteria := &warnly.ListIssueMetricsCriteria{
		ProjectIDs: []int{alert.ProjectID},
		GroupIDs:   []int64{}, // Empty means all issues in the project
		From:       from,
		To:         now,
	}

	metrics, err := w.analyticsStore.ListIssueMetrics(ctx, criteria)
	if err != nil {
		return fmt.Errorf("get issue metrics: %w", err)
	}

	triggered := false
	for _, metric := range metrics {
		var value uint64
		switch alert.Condition {
		case warnly.AlertConditionOccurrences:
			value = metric.TimesSeen
		case warnly.AlertConditionUsers:
			value = metric.UserCount
		default:
			value = metric.TimesSeen
		}

		if value > uint64(alert.Threshold) {
			triggered = true
			w.logger.Info("alert triggered",
				slog.Int("alert_id", alert.ID),
				slog.String("alert_name", alert.RuleName),
				slog.Uint64("issue_id", metric.GID),
				slog.Uint64("value", value),
				slog.Int("threshold", alert.Threshold),
			)
			break
		}
	}

	if triggered {
		alert.Status = warnly.AlertStatusTriggered
		alert.UpdatedAt = now
		t := now
		alert.LastTriggeredAt = &t

		if err := w.alertStore.UpdateAlert(ctx, alert); err != nil {
			return fmt.Errorf("update alert status: %w", err)
		}

		w.logger.Warn("ALERT NOTIFICATION would be sent here",
			slog.Int("alert_id", alert.ID),
			slog.String("alert_name", alert.RuleName),
			slog.Int("project_id", alert.ProjectID),
		)
	}

	return nil
}
