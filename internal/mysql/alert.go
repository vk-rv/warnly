package mysql

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AlertStore implements warnly.AlertStore interface.
type AlertStore struct {
	db ExtendedDB
}

// NewAlertStore is a constructor of AlertStore.
func NewAlertStore(db ExtendedDB) *AlertStore {
	return &AlertStore{db: db}
}

// ListAlerts returns a list of alerts for the given criteria.
func (s *AlertStore) ListAlerts(ctx context.Context, req *warnly.ListAlertsRequest) ([]warnly.Alert, error) {
	return []warnly.Alert{}, nil
}
