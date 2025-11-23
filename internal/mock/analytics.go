package mock

import (
	"context"
	"time"

	"github.com/vk-rv/warnly/internal/warnly"
)

// AnalyticsStore is a mock implementation of warnly.AnalyticsStore.
//
//nolint:lll // ignore
type AnalyticsStore struct {
	CalculateEventsFn       func(ctx context.Context, criteria *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error)
	CalculateFieldsFn       func(ctx context.Context, criteria warnly.FieldsCriteria) ([]warnly.TagCount, error)
	CountFieldsFn           func(ctx context.Context, criteria *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error)
	ListEventsFn            func(ctx context.Context, criteria *warnly.EventCriteria) ([]warnly.EventEntry, error)
	CountEventsFn           func(ctx context.Context, criteria *warnly.EventCriteria) (uint64, error)
	ListIssueMetricsFn      func(ctx context.Context, criteria *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error)
	CalculateEventsPerDayFn func(ctx context.Context, criteria *warnly.EventDefCriteria) ([]warnly.EventPerDay, error)
	GetIssueEventFn         func(ctx context.Context, criteria *warnly.EventDefCriteria) (*warnly.IssueEvent, error)
	ListSlowQueriesFn       func(ctx context.Context) ([]warnly.SQLQuery, error)
	ListSchemasFn           func(ctx context.Context) ([]warnly.Schema, error)
	ListErrorsFn            func(ctx context.Context, criteria warnly.ListErrorsCriteria) ([]warnly.AnalyticsStoreErr, error)
	StoreEventFn            func(ctx context.Context, event *warnly.EventClickhouse) error
	ListFieldFiltersFn      func(ctx context.Context, criteria *warnly.FieldFilterCriteria) ([]warnly.Filter, error)
	ListPopularTagsFn       func(ctx context.Context, criteria *warnly.ListPopularTagsCriteria) ([]warnly.TagCount, error)
	ListTagValuesFn         func(ctx context.Context, criteria *warnly.ListTagValuesCriteria) ([]warnly.TagValueCount, error)
	GetFilteredGroupIDsFn   func(ctx context.Context, tokens []warnly.QueryToken, from, to time.Time, projectIDs []int) ([]int64, error)
	GetEventPaginationFn    func(ctx context.Context, c *warnly.EventPaginationCriteria) (*warnly.EventPagination, error)
}

func (m *AnalyticsStore) CalculateEvents(
	ctx context.Context,
	criteria *warnly.ListIssueMetricsCriteria,
) ([]warnly.EventsPerHour, error) {
	return m.CalculateEventsFn(ctx, criteria)
}

func (m *AnalyticsStore) CalculateFields(
	ctx context.Context,
	criteria warnly.FieldsCriteria,
) ([]warnly.TagCount, error) {
	return m.CalculateFieldsFn(ctx, criteria)
}

func (m *AnalyticsStore) CountFields(
	ctx context.Context,
	criteria *warnly.EventDefCriteria,
) ([]warnly.FieldValueNum, error) {
	return m.CountFieldsFn(ctx, criteria)
}

func (m *AnalyticsStore) ListEvents(
	ctx context.Context,
	criteria *warnly.EventCriteria,
) ([]warnly.EventEntry, error) {
	return m.ListEventsFn(ctx, criteria)
}

func (m *AnalyticsStore) CountEvents(
	ctx context.Context,
	criteria *warnly.EventCriteria,
) (uint64, error) {
	return m.CountEventsFn(ctx, criteria)
}

func (m *AnalyticsStore) ListIssueMetrics(
	ctx context.Context,
	criteria *warnly.ListIssueMetricsCriteria,
) ([]warnly.IssueMetrics, error) {
	return m.ListIssueMetricsFn(ctx, criteria)
}

func (m *AnalyticsStore) CalculateEventsPerDay(
	ctx context.Context,
	criteria *warnly.EventDefCriteria,
) ([]warnly.EventPerDay, error) {
	return m.CalculateEventsPerDayFn(ctx, criteria)
}

func (m *AnalyticsStore) GetIssueEvent(
	ctx context.Context,
	criteria *warnly.EventDefCriteria,
) (*warnly.IssueEvent, error) {
	return m.GetIssueEventFn(ctx, criteria)
}

func (m *AnalyticsStore) ListSlowQueries(ctx context.Context) ([]warnly.SQLQuery, error) {
	return m.ListSlowQueriesFn(ctx)
}

func (m *AnalyticsStore) ListSchemas(ctx context.Context) ([]warnly.Schema, error) {
	return m.ListSchemasFn(ctx)
}

func (m *AnalyticsStore) ListErrors(
	ctx context.Context,
	criteria warnly.ListErrorsCriteria,
) ([]warnly.AnalyticsStoreErr, error) {
	return m.ListErrorsFn(ctx, criteria)
}

func (m *AnalyticsStore) StoreEvent(
	ctx context.Context,
	event *warnly.EventClickhouse,
) error {
	return m.StoreEventFn(ctx, event)
}

func (m *AnalyticsStore) ListFieldFilters(
	ctx context.Context,
	criteria *warnly.FieldFilterCriteria,
) ([]warnly.Filter, error) {
	return m.ListFieldFiltersFn(ctx, criteria)
}

func (m *AnalyticsStore) ListPopularTags(
	ctx context.Context,
	criteria *warnly.ListPopularTagsCriteria,
) ([]warnly.TagCount, error) {
	return m.ListPopularTagsFn(ctx, criteria)
}

func (m *AnalyticsStore) ListTagValues(
	ctx context.Context,
	criteria *warnly.ListTagValuesCriteria,
) ([]warnly.TagValueCount, error) {
	return m.ListTagValuesFn(ctx, criteria)
}

func (m *AnalyticsStore) GetFilteredGroupIDs(
	ctx context.Context,
	tokens []warnly.QueryToken,
	from,
	to time.Time,
	projectIDs []int,
) ([]int64, error) {
	return m.GetFilteredGroupIDsFn(ctx, tokens, from, to, projectIDs)
}

func (m *AnalyticsStore) GetEventPagination(
	ctx context.Context,
	c *warnly.EventPaginationCriteria,
) (*warnly.EventPagination, error) {
	if m.GetEventPaginationFn != nil {
		return m.GetEventPaginationFn(ctx, c)
	}
	return &warnly.EventPagination{}, nil
}
