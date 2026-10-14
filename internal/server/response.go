package server

import "github.com/vk-rv/warnly/internal/warnly"

// API response envelopes use explicit JSON names, including nested domain models.
type errorResponse struct {
	Error string `json:"error"`
}

type loginResponse struct {
	AuthURL      string `json:"auth_url"`
	ProviderName string `json:"provider_name"`
	IsDemo       bool   `json:"is_demo"`
}

type alertFormResponse struct {
	Alert    *warnly.Alert    `json:"alert,omitempty"`
	Projects []warnly.Project `json:"projects"`
}

type settingsResponse struct {
	User    *warnly.User                    `json:"user"`
	Webhook *warnly.WebhookConfigWithSecret `json:"webhook"`
}
