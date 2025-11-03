// Package worker provides background workers for processing alerts.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vk-rv/warnly/internal/notifier"
	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertWorker processes alert rules in the background.
//
//nolint:govet // dont care about it for now
type AlertWorker struct {
	alertStore        warnly.AlertStore
	analyticsStore    warnly.AnalyticsStore
	issueStore        warnly.IssueStore
	notificationStore warnly.NotificationStore
	webhookNotifier   *notifier.WebhookNotifier
	logger            *slog.Logger
	stopCh            chan struct{}
	now               func() time.Time
	interval          time.Duration
	instanceID        string
	lockDuration      time.Duration
	mu                sync.Mutex
	running           bool
}

// NewAlertWorker creates a new alert worker.
func NewAlertWorker(
	alertStore warnly.AlertStore,
	analyticsStore warnly.AnalyticsStore,
	issueStore warnly.IssueStore,
	notificationStore warnly.NotificationStore,
	webhookNotifier *notifier.WebhookNotifier,
	now func() time.Time,
	interval time.Duration,
	instanceID string,
	logger *slog.Logger,
) *AlertWorker {
	return &AlertWorker{
		now:               now,
		alertStore:        alertStore,
		analyticsStore:    analyticsStore,
		issueStore:        issueStore,
		notificationStore: notificationStore,
		webhookNotifier:   webhookNotifier,
		logger:            logger,
		interval:          interval,
		instanceID:        instanceID,
		lockDuration:      5 * time.Minute,
		stopCh:            make(chan struct{}),
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
	if err := w.notificationStore.CleanupExpiredLocks(ctx, w.now().UTC()); err != nil {
		w.logger.Error("process alerts: failed to cleanup expired locks", slog.Any("error", err))
		return
	}

	alerts, _, err := w.alertStore.ListAlerts(ctx, []int{}, "", 0, 1000)
	if err != nil {
		w.logger.Error("process alerts: failed to list alerts", slog.Any("error", err))
		return
	}

	alertsToProcess := make([]warnly.Alert, 0)
	for i := range alerts {
		if alerts[i].Status == warnly.AlertStatusActive || alerts[i].Status == warnly.AlertStatusTriggered {
			alertsToProcess = append(alertsToProcess, alerts[i])
		}
	}

	if len(alertsToProcess) == 0 {
		return
	}

	for i := range alertsToProcess {
		if err := w.checkAlert(ctx, &alertsToProcess[i]); err != nil {
			w.logger.Error("failed to check alert",
				slog.Int("alert_id", alertsToProcess[i].ID),
				slog.String("alert_name", alertsToProcess[i].RuleName),
				slog.Any("error", err),
			)
		}
	}
}

// checkAlert checks if an alert should be triggered or resolved.
func (w *AlertWorker) checkAlert(ctx context.Context, alert *warnly.Alert) error {
	now := w.now().UTC()
	lock := &warnly.AlertLock{
		AlertID:    alert.ID,
		InstanceID: w.instanceID,
		LockedAt:   now,
		ExpiresAt:  now.Add(w.lockDuration),
	}

	acquired, err := w.notificationStore.AcquireAlertLock(ctx, lock)
	if err != nil {
		return fmt.Errorf("check alert: acquire lock: %w", err)
	}

	if !acquired {
		w.logger.Debug("alert is locked by another instance, skipping",
			slog.Int("alert_id", alert.ID),
		)
		return nil
	}

	defer func() {
		if err := w.notificationStore.ReleaseAlertLock(ctx, alert.ID, w.instanceID); err != nil {
			w.logger.Error("failed to release lock",
				slog.Int("alert_id", alert.ID),
				slog.Any("error", err),
			)
		}
	}()

	timeframe := alert.GetTimeframeDuration()
	from := now.Add(-timeframe)

	issues, err := w.issueStore.ListIssues(ctx, &warnly.ListIssuesCriteria{
		ProjectIDs: []int{alert.ProjectID},
		From:       from,
		To:         now,
	})
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		if alert.Status == warnly.AlertStatusTriggered {
			return w.resolveAlert(ctx, alert, now)
		}
		return nil
	}

	issueIDs := make([]int64, len(issues))
	for i := range issues {
		issueIDs[i] = issues[i].ID
	}

	criteria := &warnly.ListIssueMetricsCriteria{
		ProjectIDs: []int{alert.ProjectID},
		GroupIDs:   issueIDs,
		From:       from,
		To:         now,
	}

	metrics, err := w.analyticsStore.ListIssueMetrics(ctx, criteria)
	if err != nil {
		return err
	}

	triggered := false
	for i := range metrics {
		var value uint64
		switch alert.Condition {
		case warnly.AlertConditionOccurrences:
			value = metrics[i].TimesSeen
		case warnly.AlertConditionUsers:
			value = metrics[i].UserCount
		default:
			value = metrics[i].TimesSeen
		}

		if value > uint64(alert.Threshold) {
			triggered = true
			break
		}
	}

	if triggered && alert.Status == warnly.AlertStatusActive {
		return w.triggerAlert(ctx, alert, now)
	} else if !triggered && alert.Status == warnly.AlertStatusTriggered {
		return w.resolveAlert(ctx, alert, now)
	}

	return nil
}

// triggerAlert transitions an alert to triggered state and sends notification.
func (w *AlertWorker) triggerAlert(ctx context.Context, alert *warnly.Alert, now time.Time) error {
	alert.Status = warnly.AlertStatusTriggered
	alert.UpdatedAt = now
	alert.LastTriggeredAt = &now

	if err := w.alertStore.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("update alert status: %w", err)
	}

	return w.sendNotifications(ctx, alert, warnly.AlertNotificationTriggered)
}

// resolveAlert transitions an alert to resolved state and sends notification.
func (w *AlertWorker) resolveAlert(ctx context.Context, alert *warnly.Alert, now time.Time) error {
	alert.Status = warnly.AlertStatusActive
	alert.UpdatedAt = now
	resolvedAt := now
	alert.ResolvedAt = &resolvedAt

	if err := w.alertStore.UpdateAlert(ctx, alert); err != nil {
		return err
	}

	return w.sendNotifications(ctx, alert, warnly.AlertNotificationResolved)
}

// sendNotifications sends notifications to all enabled channels for the team.
func (w *AlertWorker) sendNotifications(
	ctx context.Context,
	alert *warnly.Alert,
	notificationType warnly.AlertNotificationType,
) error {
	channels, err := w.notificationStore.ListNotificationChannels(ctx, alert.TeamID)
	if err != nil {
		return fmt.Errorf("list notification channels: %w", err)
	}

	for i := range channels {
		if !channels[i].Enabled {
			continue
		}

		notification := &warnly.AlertNotification{
			CreatedAt:        w.now().UTC(),
			AlertID:          alert.ID,
			ChannelID:        channels[i].ID,
			NotificationType: notificationType,
			Status:           warnly.AlertNotificationPending,
		}

		if err := w.notificationStore.CreateAlertNotification(ctx, notification); err != nil {
			w.logger.Error("failed to create notification record",
				slog.Int("alert_id", alert.ID),
				slog.Int("channel_id", channels[i].ID),
				slog.Any("error", err),
			)
			continue
		}

		if err := w.sendNotification(ctx, alert, &channels[i], notificationType, notification); err != nil {
			w.logger.Error("failed to send notification",
				slog.Int("alert_id", alert.ID),
				slog.Int("channel_id", channels[i].ID),
				slog.String("channel_type", string(channels[i].ChannelType)),
				slog.Any("error", err),
			)

			notification.Status = warnly.AlertNotificationFailed
			notification.ErrorMessage = err.Error()
		} else {
			notification.Status = warnly.AlertNotificationSent
			now := w.now().UTC()
			notification.SentAt = &now
		}

		if err := w.notificationStore.UpdateAlertNotification(ctx, notification); err != nil {
			w.logger.Error("failed to update notification record",
				slog.Int64("notification_id", notification.ID),
				slog.Any("error", err),
			)
		}
	}

	return nil
}

// sendNotification sends a notification to a specific channel.
func (w *AlertWorker) sendNotification(
	ctx context.Context,
	alert *warnly.Alert,
	channel *warnly.NotificationChannel,
	notificationType warnly.AlertNotificationType,
	_ *warnly.AlertNotification,
) error {
	switch channel.ChannelType {
	case warnly.NotificationChannelWebhook:
		config, err := w.notificationStore.GetWebhookConfig(ctx, channel.ID)
		if err != nil {
			return fmt.Errorf("get webhook config: %w", err)
		}

		if config.VerifiedAt == nil {
			return errors.New("webhook not verified")
		}

		if notificationType == warnly.AlertNotificationTriggered {
			return w.webhookNotifier.SendAlertTriggered(ctx, alert, config)
		}
		return w.webhookNotifier.SendAlertResolved(ctx, alert, config)

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.ChannelType)
	}
}
