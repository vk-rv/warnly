package warnly

import (
	"context"
)

// DefaultMessage is used when we can't get the error message from stacktrace.
const DefaultMessage = "(No error message)"

// EventService defines the interface for event-related operations.
type EventService interface {
	// IngestEvent ingests and stores a new event in both OLTP and OLAP databases.
	IngestEvent(ctx context.Context, req IngestRequest) (IngestEventResult, error)
}

type IngestEventResult struct {
	// EventID is the ID of the ingested event.
	EventID string
}

// IngestRequest is a request to ingest a new event.
type IngestRequest struct {
	Event     *EventBody
	IP        string
	ProjectID int
}
