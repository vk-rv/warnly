package warnly

import (
	"context"
	"time"
)

// NotificationChannelType represents the type of notification channel.
type NotificationChannelType string

const (
	// NotificationChannelWebhook represents Webhook notification channel.
	NotificationChannelWebhook NotificationChannelType = "webhook"
)

// NotificationChannel represents a notification channel configuration.
type NotificationChannel struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	ChannelType NotificationChannelType
	TeamID      int
	ID          int
	Enabled     bool
}

// WebhookConfig represents webhook configuration.
type WebhookConfig struct {
	CreatedAt       time.Time
	UpdatedAt       time.Time
	VerifiedAt      *time.Time
	URL             string
	SecretEncrypted string
	ID              int
	ChannelID       int
}

// AlertNotificationType represents the type of alert notification.
type AlertNotificationType string

const (
	// AlertNotificationTriggered represents a triggered alert notification.
	AlertNotificationTriggered AlertNotificationType = "triggered"
	// AlertNotificationResolved represents a resolved alert notification.
	AlertNotificationResolved AlertNotificationType = "resolved"
)

// AlertNotificationStatus represents the status of alert notification.
type AlertNotificationStatus string

const (
	// AlertNotificationPending represents a pending notification.
	AlertNotificationPending AlertNotificationStatus = "pending"
	// AlertNotificationSent represents a sent notification.
	AlertNotificationSent AlertNotificationStatus = "sent"
	// AlertNotificationFailed represents a failed notification.
	AlertNotificationFailed AlertNotificationStatus = "failed"
)

// AlertNotification represents an alert notification record.
type AlertNotification struct {
	CreatedAt        time.Time
	SentAt           *time.Time
	NotificationType AlertNotificationType
	Status           AlertNotificationStatus
	ErrorMessage     string
	ID               int64
	AlertID          int
	ChannelID        int
}

// AlertLock represents a distributed lock for alert processing.
type AlertLock struct {
	LockedAt   time.Time
	ExpiresAt  time.Time
	InstanceID string
	AlertID    int
}

// NotificationStore encapsulates the notification storage.
//
//nolint:interfacebloat // think about how to refactor this
type NotificationStore interface {
	// CreateNotificationChannel creates a new notification channel.
	CreateNotificationChannel(ctx context.Context, channel *NotificationChannel) error
	// GetNotificationChannel returns a notification channel by ID.
	GetNotificationChannel(ctx context.Context, channelID int) (*NotificationChannel, error)
	// ListNotificationChannels returns all notification channels for a team.
	ListNotificationChannels(ctx context.Context, teamID int) ([]NotificationChannel, error)
	// UpdateNotificationChannel updates a notification channel.
	UpdateNotificationChannel(ctx context.Context, channel *NotificationChannel) error
	// DeleteNotificationChannel deletes a notification channel.
	DeleteNotificationChannel(ctx context.Context, channelID int) error

	// CreateWebhookConfig creates a new webhook configuration.
	CreateWebhookConfig(ctx context.Context, config *WebhookConfig) error
	// GetWebhookConfig returns a webhook configuration by channel ID.
	GetWebhookConfig(ctx context.Context, channelID int) (*WebhookConfig, error)
	// UpdateWebhookConfig updates a webhook configuration.
	UpdateWebhookConfig(ctx context.Context, config *WebhookConfig) error

	// CreateAlertNotification creates a new alert notification record.
	CreateAlertNotification(ctx context.Context, notification *AlertNotification) error
	// UpdateAlertNotification updates an alert notification record.
	UpdateAlertNotification(ctx context.Context, notification *AlertNotification) error
	// ListPendingNotifications returns all pending notifications.
	ListPendingNotifications(ctx context.Context, limit int) ([]AlertNotification, error)

	// AcquireAlertLock attempts to acquire a lock for processing an alert.
	AcquireAlertLock(ctx context.Context, lock *AlertLock) (bool, error)
	// ReleaseAlertLock releases a lock for an alert.
	ReleaseAlertLock(ctx context.Context, alertID int, instanceID string) error
	// CleanupExpiredLocks removes expired locks.
	CleanupExpiredLocks(ctx context.Context, now time.Time) error
}

// NotificationService encapsulates service domain logic.
type NotificationService interface {
	// SaveWebhookConfig saves or updates webhook configuration for a team.
	SaveWebhookConfig(ctx context.Context, req *SaveWebhookConfigRequest) error
	// TestWebhook sends a test notification to the configured webhook.
	TestWebhook(ctx context.Context, req *TestWebhookRequest) error
	// GetWebhookConfigWithSecretByTeamID returns the webhook configuration with decrypted secret for a team.
	GetWebhookConfigWithSecretByTeamID(ctx context.Context, teamID int) (*WebhookConfigWithSecret, error)
}

// WebhookConfigWithSecret holds webhook config with decrypted secret.
type WebhookConfigWithSecret struct {
	URL    string
	Secret string
}

// SaveWebhookConfigRequest is a request to save or update webhook configuration.
type SaveWebhookConfigRequest struct {
	User   *User
	URL    string
	Secret string
	TeamID int
}

// TestWebhookRequest is a request to send a test notification to the configured webhook.
type TestWebhookRequest struct {
	User   *User
	TeamID int
}
