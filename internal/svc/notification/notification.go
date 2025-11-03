// Package notification provides the implementation of the warnly.NotificationService interface.
package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vk-rv/warnly/internal/notifier"
	"github.com/vk-rv/warnly/internal/warnly"
)

// NotificationService implements notification management logic.
type NotificationService struct {
	notificationStore warnly.NotificationStore
	teamStore         warnly.TeamStore
	webhookNotifier   *notifier.WebhookNotifier
	now               func() time.Time
	logger            *slog.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notificationStore warnly.NotificationStore,
	teamStore warnly.TeamStore,
	webhookNotifier *notifier.WebhookNotifier,
	now func() time.Time,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		notificationStore: notificationStore,
		teamStore:         teamStore,
		webhookNotifier:   webhookNotifier,
		now:               now,
		logger:            logger,
	}
}

// SaveWebhookConfig saves or updates webhook configuration for a team.
//
//nolint:gocyclo,cyclop // later
func (s *NotificationService) SaveWebhookConfig(
	ctx context.Context,
	req *warnly.SaveWebhookConfigRequest,
) error {
	teams, err := s.teamStore.ListTeams(ctx, int(req.User.ID))
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	hasAccess := false
	for i := range teams {
		if teams[i].ID == req.TeamID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		return warnly.ErrNotFound
	}

	now := s.now().UTC()

	if req.URL == "" {
		channels, err := s.notificationStore.ListNotificationChannels(ctx, req.TeamID)
		if err != nil {
			return err
		}
		for i := range channels {
			if channels[i].ChannelType != warnly.NotificationChannelWebhook {
				continue
			}
			webhookConfig, err := s.notificationStore.GetWebhookConfig(ctx, channels[i].ID)
			if err != nil {
				if errors.Is(err, warnly.ErrNotFound) {
					return nil
				}
				return err
			}
			webhookConfig.UpdatedAt = now
			webhookConfig.URL = ""
			webhookConfig.SecretEncrypted = ""
			if err := s.notificationStore.UpdateWebhookConfig(ctx, webhookConfig); err != nil {
				return err
			}
			break
		}
		return nil
	}

	channels, err := s.notificationStore.ListNotificationChannels(ctx, req.TeamID)
	if err != nil {
		return err
	}

	var channel *warnly.NotificationChannel
	for i := range channels {
		if channels[i].ChannelType == warnly.NotificationChannelWebhook {
			channel = &channels[i]
			break
		}
	}

	if channel == nil {
		channel = &warnly.NotificationChannel{
			CreatedAt:   now,
			UpdatedAt:   now,
			TeamID:      req.TeamID,
			Name:        "Default Webhook",
			ChannelType: warnly.NotificationChannelWebhook,
			Enabled:     true,
		}
		if err := s.notificationStore.CreateNotificationChannel(ctx, channel); err != nil {
			return err
		}
	}

	var encryptedSecret string
	if req.Secret != "" {
		encryptedSecret, err = s.webhookNotifier.EncryptSecret(req.Secret)
		if err != nil {
			return err
		}
	}

	webhookConfig, err := s.notificationStore.GetWebhookConfig(ctx, channel.ID)
	if err != nil {
		if !errors.Is(err, warnly.ErrNotFound) {
			return err
		}
		webhookConfig = &warnly.WebhookConfig{
			CreatedAt:       now,
			UpdatedAt:       now,
			ChannelID:       channel.ID,
			URL:             req.URL,
			SecretEncrypted: encryptedSecret,
		}
		if err := s.notificationStore.CreateWebhookConfig(ctx, webhookConfig); err != nil {
			return err
		}
	} else {
		webhookConfig.UpdatedAt = now
		webhookConfig.URL = req.URL
		webhookConfig.SecretEncrypted = encryptedSecret
		webhookConfig.VerifiedAt = &now

		if err := s.notificationStore.UpdateWebhookConfig(ctx, webhookConfig); err != nil {
			return err
		}
	}

	if err := s.webhookNotifier.SendWebhook(ctx, webhookConfig, &notifier.AlertPayload{
		AlertID:      0,
		AlertName:    "Test Alert",
		ProjectID:    0,
		TeamID:       req.TeamID,
		Status:       "test",
		Threshold:    100,
		Condition:    "occurrences",
		Timeframe:    "1h",
		HighPriority: false,
		Timestamp:    s.now().UTC(),
	}); err != nil {
		return fmt.Errorf("send test webhook: %w", err)
	}

	return nil
}

// GetWebhookConfigByTeamID returns the webhook configuration for a team.
func (s *NotificationService) GetWebhookConfigByTeamID(ctx context.Context, teamID int) (*warnly.WebhookConfig, error) {
	channels, err := s.notificationStore.ListNotificationChannels(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}

	var channel *warnly.NotificationChannel
	for i := range channels {
		if channels[i].ChannelType == warnly.NotificationChannelWebhook {
			channel = &channels[i]
			break
		}
	}

	if channel == nil {
		return nil, warnly.ErrNotFound
	}

	webhookConfig, err := s.notificationStore.GetWebhookConfig(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("get webhook config: %w", err)
	}

	return webhookConfig, nil
}

// GetWebhookConfigWithSecretByTeamID returns the webhook configuration with decrypted secret for a team.
func (s *NotificationService) GetWebhookConfigWithSecretByTeamID(
	ctx context.Context,
	teamID int,
) (*warnly.WebhookConfigWithSecret, error) {
	config, err := s.GetWebhookConfigByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, warnly.ErrNotFound) {
			return &warnly.WebhookConfigWithSecret{}, nil
		}
		return nil, err
	}

	var secret string
	if config.SecretEncrypted != "" {
		secret, err = s.webhookNotifier.DecryptSecret(config.SecretEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt secret: %w", err)
		}
	}

	return &warnly.WebhookConfigWithSecret{
		URL:    config.URL,
		Secret: secret,
	}, nil
}
