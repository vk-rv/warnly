// Package notifier implements webhook notifications.
package notifier

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// WebhookNotifier sends notifications via HTTP webhooks.
type WebhookNotifier struct {
	store         warnly.NotificationStore
	now           func() time.Time
	logger        *slog.Logger
	httpClient    *http.Client
	encryptionKey []byte
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(
	store warnly.NotificationStore,
	encryptionKey []byte,
	httpClient *http.Client,
	now func() time.Time,
	logger *slog.Logger,
) *WebhookNotifier {
	return &WebhookNotifier{
		store:         store,
		logger:        logger,
		now:           now,
		encryptionKey: deriveKey(encryptionKey),
		httpClient:    httpClient,
	}
}

// deriveKey derives a 32-byte key from the input using SHA-256.
func deriveKey(key []byte) []byte {
	hash := sha256.Sum256(key)
	return hash[:]
}

// EncryptSecret encrypts a webhook secret using AES-GCM.
func (wn *WebhookNotifier) EncryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}

	block, err := aes.NewCipher(wn.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts a webhook secret using AES-GCM.
func (wn *WebhookNotifier) DecryptSecret(encryptedSecret string) (string, error) {
	if encryptedSecret == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(wn.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// AlertPayload represents the webhook payload for alert notifications.
type AlertPayload struct {
	Timestamp    time.Time `json:"timestamp"`
	AlertName    string    `json:"alert_name"`
	Status       string    `json:"status"`
	Condition    string    `json:"condition"`
	Timeframe    string    `json:"timeframe"`
	AlertID      int       `json:"alert_id"`
	ProjectID    int       `json:"project_id"`
	TeamID       int       `json:"team_id"`
	Threshold    int       `json:"threshold"`
	HighPriority bool      `json:"high_priority"`
}

// SendWebhook sends a webhook notification.
func (wn *WebhookNotifier) SendWebhook(ctx context.Context, config *warnly.WebhookConfig, payload *AlertPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook notifier: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("webhook notifier: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if config.SecretEncrypted != "" {
		secret, err := wn.DecryptSecret(config.SecretEncrypted)
		if err != nil {
			return fmt.Errorf("webhook notifier: decrypt secret: %w", err)
		}
		signature := computeHMAC(jsonData, []byte(secret))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := wn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webhook notifier: read response body: %w", err)
		}
		return fmt.Errorf("webhook notifier: webhook returned non-2xx status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendAlertTriggered sends an alert triggered notification.
func (wn *WebhookNotifier) SendAlertTriggered(ctx context.Context, alert *warnly.Alert, config *warnly.WebhookConfig) error {
	payload := &AlertPayload{
		AlertID:      alert.ID,
		AlertName:    alert.RuleName,
		ProjectID:    alert.ProjectID,
		TeamID:       alert.TeamID,
		Status:       "triggered",
		Threshold:    alert.Threshold,
		Condition:    getConditionName(alert.Condition),
		Timeframe:    getTimeframeName(alert.Timeframe),
		HighPriority: alert.HighPriority,
		Timestamp:    wn.now().UTC(),
	}

	return wn.SendWebhook(ctx, config, payload)
}

// SendAlertResolved sends an alert resolved notification.
func (wn *WebhookNotifier) SendAlertResolved(ctx context.Context, alert *warnly.Alert, config *warnly.WebhookConfig) error {
	payload := &AlertPayload{
		AlertID:      alert.ID,
		AlertName:    alert.RuleName,
		ProjectID:    alert.ProjectID,
		TeamID:       alert.TeamID,
		Status:       "resolved",
		Threshold:    alert.Threshold,
		Condition:    getConditionName(alert.Condition),
		Timeframe:    getTimeframeName(alert.Timeframe),
		HighPriority: alert.HighPriority,
		Timestamp:    wn.now().UTC(),
	}

	return wn.SendWebhook(ctx, config, payload)
}

func getConditionName(condition warnly.AlertCondition) string {
	switch condition {
	case warnly.AlertConditionOccurrences:
		return "occurrences"
	case warnly.AlertConditionUsers:
		return "users_affected"
	default:
		return "unknown"
	}
}

func getTimeframeName(timeframe warnly.AlertTimeframe) string {
	switch timeframe {
	case warnly.AlertTimeframe1Min:
		return "1m"
	case warnly.AlertTimeframe5Min:
		return "5m"
	case warnly.AlertTimeframe15Min:
		return "15m"
	case warnly.AlertTimeframe1Hour:
		return "1h"
	case warnly.AlertTimeframe1Day:
		return "1d"
	case warnly.AlertTimeframe1Week:
		return "1w"
	case warnly.AlertTimeframe30Days:
		return "30d"
	default:
		return "unknown"
	}
}

// computeHMAC computes HMAC-SHA256 signature.
func computeHMAC(message, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return hex.EncodeToString(h.Sum(nil))
}
