package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// NotificationStore implements warnly.NotificationStore.
type NotificationStore struct {
	db ExtendedDB
}

// NewNotificationStore creates a new NotificationStore.
func NewNotificationStore(db ExtendedDB) *NotificationStore {
	return &NotificationStore{db: db}
}

// CreateNotificationChannel creates a new notification channel.
func (s *NotificationStore) CreateNotificationChannel(ctx context.Context, channel *warnly.NotificationChannel) error {
	const query = `
		INSERT INTO notification_channel (created_at, updated_at, team_id, name, channel_type, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(
		ctx,
		query,
		channel.CreatedAt,
		channel.UpdatedAt,
		channel.TeamID,
		channel.Name,
		channel.ChannelType,
		channel.Enabled)
	if err != nil {
		return fmt.Errorf("mysql: insert notification channel: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql: get last insert id: %w", err)
	}

	channel.ID = int(id)

	return nil
}

// GetNotificationChannel returns a notification channel by ID.
func (s *NotificationStore) GetNotificationChannel(ctx context.Context, channelID int) (*warnly.NotificationChannel, error) {
	const query = `
		SELECT id, created_at, updated_at, team_id, name, channel_type, enabled
		FROM notification_channel
		WHERE id = ?
	`
	var channel warnly.NotificationChannel
	err := s.db.QueryRowContext(ctx, query, channelID).Scan(
		&channel.ID,
		&channel.CreatedAt,
		&channel.UpdatedAt,
		&channel.TeamID,
		&channel.Name,
		&channel.ChannelType,
		&channel.Enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("mysql: get notification channel: %w", err)
	}

	return &channel, nil
}

// ListNotificationChannels returns all notification channels for a team.
func (s *NotificationStore) ListNotificationChannels(ctx context.Context, teamID int) ([]warnly.NotificationChannel, error) {
	const query = `
		SELECT id, created_at, updated_at, team_id, name, channel_type, enabled
		FROM notification_channel
		WHERE team_id = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("mysql: list notification channels: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var channels []warnly.NotificationChannel
	for rows.Next() {
		var channel warnly.NotificationChannel
		if err := rows.Scan(
			&channel.ID,
			&channel.CreatedAt,
			&channel.UpdatedAt,
			&channel.TeamID,
			&channel.Name,
			&channel.ChannelType,
			&channel.Enabled,
		); err != nil {
			return nil, fmt.Errorf("mysql: scan notification channel: %w", err)
		}
		channels = append(channels, channel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: list notification channels: %w", err)
	}

	return channels, nil
}

// UpdateNotificationChannel updates a notification channel.
func (s *NotificationStore) UpdateNotificationChannel(ctx context.Context, channel *warnly.NotificationChannel) error {
	const query = `
		UPDATE notification_channel
		SET updated_at = ?, name = ?, enabled = ?
		WHERE id = ?
	`
	res, err := s.db.ExecContext(ctx, query, channel.UpdatedAt, channel.Name, channel.Enabled, channel.ID)
	if err != nil {
		return fmt.Errorf("mysql: update notification channel: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mysql: update notification channel: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("mysql: update notification channel: bad rows affected : %d", rowsAffected)
	}

	return nil
}

// DeleteNotificationChannel deletes a notification channel.
func (s *NotificationStore) DeleteNotificationChannel(ctx context.Context, channelID int) error {
	const query = `DELETE FROM notification_channel WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, channelID)
	if err != nil {
		return fmt.Errorf("mysql: delete notification channel: %w", err)
	}

	return nil
}

// CreateWebhookConfig creates a new webhook configuration.
func (s *NotificationStore) CreateWebhookConfig(ctx context.Context, config *warnly.WebhookConfig) error {
	const query = `
		INSERT INTO webhook_config (created_at, updated_at, channel_id, url, secret_encrypted, verified_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(
		ctx,
		query,
		config.CreatedAt,
		config.UpdatedAt,
		config.ChannelID,
		config.URL,
		config.SecretEncrypted,
		config.VerifiedAt)
	if err != nil {
		return fmt.Errorf("mysql: insert webhook config: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql: get last insert id: %w", err)
	}

	config.ID = int(id)

	return nil
}

// GetWebhookConfig returns a webhook configuration by channel ID.
func (s *NotificationStore) GetWebhookConfig(ctx context.Context, channelID int) (*warnly.WebhookConfig, error) {
	const query = `
		SELECT id, created_at, updated_at, channel_id, url, secret_encrypted, verified_at
		FROM webhook_config
		WHERE channel_id = ?
	`
	var (
		config          warnly.WebhookConfig
		secretEncrypted sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, channelID).Scan(
		&config.ID,
		&config.CreatedAt,
		&config.UpdatedAt,
		&config.ChannelID,
		&config.URL,
		&secretEncrypted,
		&config.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, warnly.ErrNotFound
		}
		return nil, fmt.Errorf("mysql: get webhook config: %w", err)
	}

	if secretEncrypted.Valid {
		config.SecretEncrypted = secretEncrypted.String
	}

	return &config, nil
}

// UpdateWebhookConfig updates a webhook configuration.
func (s *NotificationStore) UpdateWebhookConfig(ctx context.Context, config *warnly.WebhookConfig) error {
	const query = `
		UPDATE webhook_config
		SET updated_at = ?, url = ?, secret_encrypted = ?, verified_at = ?
		WHERE channel_id = ?
	`
	_, err := s.db.ExecContext(
		ctx,
		query,
		config.UpdatedAt,
		config.URL,
		config.SecretEncrypted,
		config.VerifiedAt,
		config.ChannelID)
	if err != nil {
		return fmt.Errorf("mysql: update webhook config: %w", err)
	}

	return nil
}

// CreateAlertNotification creates a new alert notification record.
func (s *NotificationStore) CreateAlertNotification(ctx context.Context, notification *warnly.AlertNotification) error {
	const query = `
		INSERT INTO alert_notification (created_at, alert_id, channel_id, notification_type, status, error_message, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(
		ctx,
		query,
		notification.CreatedAt,
		notification.AlertID,
		notification.ChannelID,
		notification.NotificationType,
		notification.Status,
		notification.ErrorMessage,
		notification.SentAt)
	if err != nil {
		return fmt.Errorf("mysql: insert alert notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("mysql: get last insert id: %w", err)
	}

	notification.ID = id

	return nil
}

// UpdateAlertNotification updates an alert notification record.
func (s *NotificationStore) UpdateAlertNotification(ctx context.Context, notification *warnly.AlertNotification) error {
	const query = `
		UPDATE alert_notification
		SET status = ?, error_message = ?, sent_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(
		ctx,
		query,
		notification.Status,
		notification.ErrorMessage,
		notification.SentAt,
		notification.ID)
	if err != nil {
		return fmt.Errorf("mysql: update alert notification: %w", err)
	}

	return nil
}

// ListPendingNotifications returns all pending notifications.
func (s *NotificationStore) ListPendingNotifications(ctx context.Context, limit int) ([]warnly.AlertNotification, error) {
	const query = `
		SELECT id, created_at, alert_id, channel_id, notification_type, status, error_message, sent_at
		FROM alert_notification
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("mysql: query pending notifications: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var notifications []warnly.AlertNotification
	for rows.Next() {
		var n warnly.AlertNotification
		var errorMessage sql.NullString
		var sentAt sql.NullTime

		if err := rows.Scan(
			&n.ID,
			&n.CreatedAt,
			&n.AlertID,
			&n.ChannelID,
			&n.NotificationType,
			&n.Status,
			&errorMessage,
			&sentAt,
		); err != nil {
			return nil, fmt.Errorf("mysql: scan alert notification: %w", err)
		}

		if errorMessage.Valid {
			n.ErrorMessage = errorMessage.String
		}
		if sentAt.Valid {
			n.SentAt = &sentAt.Time
		}

		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// AcquireAlertLock attempts to acquire a lock for processing an alert.
func (s *NotificationStore) AcquireAlertLock(ctx context.Context, lock *warnly.AlertLock) (bool, error) {
	const query = `
		INSERT INTO alert_lock (alert_id, instance_id, locked_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			instance_id = IF(expires_at < NOW(), VALUES(instance_id), instance_id),
			locked_at = IF(expires_at < NOW(), VALUES(locked_at), locked_at),
			expires_at = IF(expires_at < NOW(), VALUES(expires_at), expires_at)
	`

	_, err := s.db.ExecContext(ctx, query, lock.AlertID, lock.InstanceID, lock.LockedAt, lock.ExpiresAt)
	if err != nil {
		return false, fmt.Errorf("mysql: acquire alert lock: %w", err)
	}

	var currentInstanceID string
	checkQuery := `SELECT instance_id FROM alert_lock WHERE alert_id = ? AND expires_at > NOW()`
	err = s.db.QueryRowContext(ctx, checkQuery, lock.AlertID).Scan(&currentInstanceID)
	if err != nil {
		return false, fmt.Errorf("mysql: check alert lock: %w", err)
	}

	return currentInstanceID == lock.InstanceID, nil
}

// ReleaseAlertLock releases a lock for an alert.
func (s *NotificationStore) ReleaseAlertLock(ctx context.Context, alertID int, instanceID string) error {
	const query = `DELETE FROM alert_lock WHERE alert_id = ? AND instance_id = ?`
	_, err := s.db.ExecContext(ctx, query, alertID, instanceID)
	if err != nil {
		return fmt.Errorf("mysql: release alert lock: %w", err)
	}

	return nil
}

// CleanupExpiredLocks removes expired locks.
func (s *NotificationStore) CleanupExpiredLocks(ctx context.Context, now time.Time) error {
	const query = `DELETE FROM alert_lock WHERE expires_at < ?`
	_, err := s.db.ExecContext(ctx, query, now)
	if err != nil {
		return fmt.Errorf("mysql: cleanup expired locks: %w", err)
	}

	return nil
}
