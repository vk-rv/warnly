package warnly

import (
	"context"
	"time"
)

// AnalyticsStore encapsulate the storage of analytics data.
// Generally speaking, it can be either clickhouse or druid or any other storage.
//
//nolint:interfacebloat // think about how to refactor this
type AnalyticsStore interface {
	// CalculateEvents calculates the number of events per day split by hour.
	CalculateEvents(ctx context.Context, criteria *ListIssueMetricsCriteria) ([]EventsPerHour, error)
	// ListIssueMetrics lists issue metrics for the given project IDs and issue IDs within the specified time range.
	// It displays how many times each issue was seen, when it was first and last seen
	// and the number of unique users affected.
	ListIssueMetrics(ctx context.Context, criteria *ListIssueMetricsCriteria) ([]IssueMetrics, error)
	// CalculateEventsPerDay calculates the number of events per day for a given issue and project
	// within a specified time range.
	CalculateEventsPerDay(ctx context.Context, criteria EventDefCriteria) ([]EventPerDay, error)
	// CountFields counts additional fields for a given issue and project within a specified time range.
	CountFields(ctx context.Context, criteria EventDefCriteria) ([]FieldValueNum, error)
	// CalculateFields calculates the number of occurrences of each field for a given group and project
	// within a specified time range.
	CalculateFields(ctx context.Context, criteria FieldsCriteria) ([]TagCount, error)
	// GetIssueEvent retrieves a single event associated with a specific issue and project within a given time range.
	GetIssueEvent(ctx context.Context, criteria EventDefCriteria) (*IssueEvent, error)
	// CountEvents counts the number of events based on the given criteria.
	CountEvents(ctx context.Context, criteria *EventCriteria) (uint64, error)
	// ListEvents lists error events based on the given criteria.
	ListEvents(ctx context.Context, criteria *EventCriteria) ([]EventEntry, error)
	// ListSlowQueries lists olap slow SQL queries from the system.
	ListSlowQueries(ctx context.Context) ([]SQLQuery, error)
	// ListSchemas lists olap database schemas from largest to smallest.
	ListSchemas(ctx context.Context) ([]Schema, error)
	// ListErrors lists recent errors from olap system.
	ListErrors(ctx context.Context, criteria ListErrorsCriteria) ([]AnalyticsStoreErr, error)
	// StoreEvent stores an event in the analytics database.
	StoreEvent(ctx context.Context, event *EventClickhouse) error
}

// EventDefCriteria represents the criteria for querying events.
type EventDefCriteria struct {
	From      time.Time
	To        time.Time
	GroupID   int
	ProjectID int
}

// ListIssueMetricsCriteria represents the criteria for listing issue metrics.
type ListIssueMetricsCriteria struct {
	From       time.Time
	To         time.Time
	ProjectIDs []int
	GroupIDs   []int64
}

// ListErrorsCriteria represents the criteria for listing errors
// from the analytics store.
type ListErrorsCriteria struct {
	LastErrorTime time.Time
}

// EventEntry represents an event entry in the analytics store.
type EventEntry struct {
	CreatedAt time.Time
	EventID   string
	Title     string
	Message   string
	Release   string
	Env       string
	User      string
	OS        string
	UserID    uint64
}

// FieldsCriteria represents the criteria for querying fields.
type FieldsCriteria struct {
	From      time.Time
	To        time.Time
	IssueID   int
	ProjectID int
}

// EventCriteria represents the criteria for querying events.
type EventCriteria struct {
	From      time.Time
	To        time.Time
	Tags      map[string]QueryValue
	Message   string
	ProjectID int
	GroupID   int
	Limit     int
	Offset    int
}

// QueryValue represents a value in a query with an optional negation flag.
type QueryValue struct {
	Value string
	IsNot bool
}

// EventsPerHour represents the number of events per hour.
type EventsPerHour struct {
	TS        time.Time
	ProjectID int
	Count     int
}

// EventPerDay represents the number of events per day.
type EventPerDay struct {
	Time  time.Time
	GID   uint64
	Count uint64
}

// TagCount represents a tag and its count.
type TagCount struct {
	Tag   string
	Count uint64
}

// FieldValueNum represents a tag value and its associated metrics.
type FieldValueNum struct {
	FirstSeen       time.Time
	LastSeen        time.Time
	Tag             string
	Value           string
	Count           uint64
	PercentsOfTotal float64
}

// GroupMetrics represents the metrics of the group.
type GroupMetrics struct {
	FirstSeen time.Time
	LastSeen  time.Time
	GID       int
	TimesSeen int
	Users     int
}
