package duckdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/duckdb"
	"github.com/vk-rv/warnly/internal/migrator"
	"github.com/vk-rv/warnly/internal/warnly"
)

func testDatabase(t *testing.T) (*sql.DB, *duckdb.Store) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION=1 to run MySQL/DuckDB container tests")
	}
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)
	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "evgeniypatlan/test-images", Tag: "mysql-9.7-duckdb-v0.2.0",
		Env: []string{"MYSQL_ROOT_PASSWORD=test-root"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Purge(container)) })
	require.NoError(t, container.Expire(600))
	rootDSN := fmt.Sprintf("root:test-root@tcp(%s)/mysql?timeout=2s&readTimeout=5s", container.GetHostPort("3306/tcp"))
	root, err := sql.Open("mysql", rootDSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	pool.MaxWait = 2 * time.Minute
	require.NoError(t, pool.Retry(func() error { return root.PingContext(t.Context()) }))
	_, err = root.ExecContext(t.Context(), "CREATE DATABASE analytics")
	require.NoError(t, err)
	dsn, err := duckdb.NormalizeDSN(fmt.Sprintf("root:test-root@tcp(%s)/analytics", container.GetHostPort("3306/tcp")))
	require.NoError(t, err)
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	pool.MaxWait = 2 * time.Minute
	require.NoError(t, pool.Retry(func() error { return db.PingContext(t.Context()) }))
	connected, closeDB, err := duckdb.ConnectLoop(t.Context(), dsn, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, connected)
	require.NoError(t, closeDB())
	for range 2 {
		m, err := migrator.NewDuckDBMigrator(dsn, slog.Default())
		require.NoError(t, err)
		require.NoError(t, m.Up(false))
		sourceErr, dbErr := m.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, dbErr)
	}
	for _, table := range []string{"event", "event_tag"} {
		var engine string
		require.NoError(t, db.QueryRowContext(t.Context(), `SELECT ENGINE FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, table).Scan(&engine))
		require.Equal(t, "DuckDB", engine)
	}
	return db, duckdb.NewStore(db)
}

//nolint:paralleltest // Subtests share mutable fixtures and the server's pushdown counter.
func TestAnalyticsIntegration(t *testing.T) {
	db, store := testDatabase(t)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Hour)
	from, to := now.Add(-2*time.Hour), now.Add(2*time.Hour)
	event := func(n int) *warnly.EventClickhouse {
		return &warnly.EventClickhouse{
			EventID: fmt.Sprintf("%032x", n), ProjectID: 1, GroupID: 10,
			CreatedAt: now.Add(time.Duration(n) * time.Minute), RetentionDays: 30,
			User: "alice", UserEmail: "alice@example.com", UserName: "Alice", UserUsername: "alice",
			Message: "Database FAILURE 100%_", Title: "Timeout", Env: fixtureProd, Release: "v1",
			TagsKey: []string{"os", fixtureEnv, fixtureSpecial}, TagsValue: []string{"linux", fixtureProd, fixtureQuote},
			ContextsKey: []string{"runtime"}, ContextsValue: []string{`{"name":"go"}`},
			ExceptionFramesAbsPath: []string{"/app/main.go"}, ExceptionFramesFunction: []string{"main"},
			ExceptionFramesLineNo: []uint32{42}, ExceptionFramesColNo: []uint32{7}, ExceptionFramesInApp: warnly.Uint8Array{1},
		}
	}
	a, b, c := event(1), event(2), event(3)
	b.User = "bob"
	b.TagsValue[1] = "stage"
	c.User = ""
	c.TagsKey, c.TagsValue = nil, nil
	c.CreatedAt = b.CreatedAt // exercise stable pagination with equal timestamps
	for _, ev := range []*warnly.EventClickhouse{a, b, c} {
		require.NoError(t, store.StoreEvent(ctx, ev))
	}
	foreign, deleted, expired, boundary := event(4), event(5), event(6), event(7)
	foreign.ProjectID = 2
	deleted.Deleted = 1
	expired.CreatedAt, expired.RetentionDays = now.AddDate(0, 0, -2), 1
	boundary.CreatedAt = to
	for _, ev := range []*warnly.EventClickhouse{foreign, deleted, expired, boundary} {
		require.NoError(t, store.StoreEvent(ctx, ev))
	}
	ec := &warnly.EventCriteria{ProjectID: 1, GroupID: 10, From: from, To: to, Limit: 20}
	dc := &warnly.EventDefCriteria{ProjectID: 1, GroupID: 10, From: from, To: to}

	t.Run("round trip and idempotent ingestion", func(t *testing.T) {
		require.NoError(t, store.StoreEvent(ctx, a))
		count, err := store.CountEvents(ctx, ec)
		require.NoError(t, err)
		require.EqualValues(t, 3, count)
		criteria := *dc
		criteria.EventID = "00000000-0000-0000-0000-000000000001"
		got, err := store.GetIssueEvent(ctx, &criteria)
		require.NoError(t, err)
		require.Equal(t, a.EventID, got.EventID)
		require.Equal(t, a.TagsKey, got.TagsKey)
		require.Equal(t, a.TagsValue, got.TagsValue)
		require.Equal(t, a.ContextsValue, got.ContextsValue)
		require.Equal(t, a.ContextsKey, got.ContextsKey)
		require.Equal(t, []int{42}, got.ExceptionFramesLineno)
		require.Equal(t, []int{7}, got.ExceptionFramesColno)
		require.Equal(t, []int{1}, got.ExceptionFramesInApp)
		require.Equal(t, a.ExceptionFramesAbsPath, got.ExceptionFramesAbsPath)
		require.Equal(t, a.ExceptionFramesFunction, got.ExceptionFramesFunction)
		criteria.ProjectID = 2
		missing, err := store.GetIssueEvent(ctx, &criteria)
		require.NoError(t, err)
		require.Empty(t, missing.EventID)
		entries, err := store.ListEvents(ctx, ec)
		require.NoError(t, err)
		require.Len(t, entries, 3)
		require.Equal(t, c.EventID, entries[0].EventID)
		require.Equal(t, "linux", entries[2].OS)
		require.Equal(t, a.UserEmail, entries[2].UserEmail)
	})

	t.Run("filters and isolation", func(t *testing.T) {
		for _, test := range []struct {
			name    string
			tags    map[string]warnly.QueryValue
			message string
			count   uint64
		}{
			{"positive", map[string]warnly.QueryValue{fixtureEnv: {Value: fixtureProd}}, "", 1},
			{"negative includes missing", map[string]warnly.QueryValue{fixtureEnv: {Value: fixtureProd, IsNot: true}}, "", 2},
			{"case sensitive tags", map[string]warnly.QueryValue{fixtureEnv: {Value: "PROD"}}, "", 0},
			{"trailing spaces are significant", map[string]warnly.QueryValue{fixtureEnv: {Value: "prod "}}, "", 0},
			{"escaped", map[string]warnly.QueryValue{fixtureSpecial: {Value: fixtureQuote}}, "", 2},
			{"case insensitive literal", nil, "failure 100%_", 3},
			{"SQL injection", nil, "' OR 1=1 --", 0},
			{"combined", map[string]warnly.QueryValue{fixtureEnv: {Value: fixtureProd}}, "failure", 1},
		} {
			t.Run(test.name, func(t *testing.T) {
				criteria := *ec
				criteria.Tags, criteria.Message = test.tags, test.message
				count, err := store.CountEvents(ctx, &criteria)
				require.NoError(t, err)
				require.Equal(t, test.count, count)
				entries, err := store.ListEvents(ctx, &criteria)
				require.NoError(t, err)
				require.Len(t, entries, int(test.count))
			})
		}
		criteria := *ec
		criteria.Limit, criteria.Offset = 1, 1
		entries, err := store.ListEvents(ctx, &criteria)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, b.EventID, entries[0].EventID)
		ids, err := store.GetFilteredGroupIDs(ctx, []warnly.QueryToken{{IsRawText: true, Value: "TIMEOUT"}, {Key: fixtureEnv, Value: fixtureProd}}, from, to.Add(-time.Second), []int{1})
		require.NoError(t, err)
		require.Equal(t, []int64{10}, ids)
		ids, err = store.GetFilteredGroupIDs(ctx, nil, from, to, nil)
		require.NoError(t, err)
		require.Empty(t, ids)
		ids, err = store.GetFilteredGroupIDs(ctx, []warnly.QueryToken{{Key: fixtureEnv, Value: fixtureProd, Operator: "is not"}}, from, to, []int{1})
		require.NoError(t, err)
		require.Equal(t, []int64{10}, ids)
	})

	t.Run("aggregates and pushdown", func(t *testing.T) {
		var variable string
		var before, after uint64
		require.NoError(t, db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Ducksdb_pushdown_count'").Scan(&variable, &before))
		mc := &warnly.ListIssueMetricsCriteria{From: from, To: to.Add(-time.Second), ProjectIDs: []int{1}, GroupIDs: []int64{10}}
		metrics, err := store.ListIssueMetrics(ctx, mc)
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		require.EqualValues(t, 3, metrics[0].TimesSeen)
		require.EqualValues(t, 2, metrics[0].UserCount)
		require.WithinDuration(t, a.CreatedAt, metrics[0].FirstSeen, time.Microsecond)
		hours, err := store.CalculateEvents(ctx, mc)
		require.NoError(t, err)
		require.Len(t, hours, 1)
		require.Equal(t, 3, hours[0].Count)
		require.Equal(t, now, hours[0].TS)
		days, err := store.CalculateEventsPerDay(ctx, dc)
		require.NoError(t, err)
		require.Len(t, days, 1)
		require.EqualValues(t, 3, days[0].Count)
		require.NoError(t, db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Ducksdb_pushdown_count'").Scan(&variable, &after))
		require.Greater(t, after, before, "analytics must actually execute in DuckDB")
		mc.ProjectIDs = nil
		hours, err = store.CalculateEvents(ctx, mc)
		require.NoError(t, err)
		require.Empty(t, hours)
	})

	t.Run("tag aggregations", func(t *testing.T) {
		fields, err := store.CalculateFields(ctx, warnly.FieldsCriteria{From: from, To: to, ProjectID: 1, IssueID: 10})
		require.NoError(t, err)
		require.Len(t, fields, 3)
		for _, field := range fields {
			require.EqualValues(t, 2, field.Count)
		}
		values, err := store.CountFields(ctx, dc)
		require.NoError(t, err)
		require.Len(t, values, 4)
		filters, err := store.ListFieldFilters(ctx, &warnly.FieldFilterCriteria{From: from, To: to, ProjectIDs: []int{1}})
		require.NoError(t, err)
		require.Len(t, filters, 4)
		tags, err := store.ListPopularTags(ctx, &warnly.ListPopularTagsCriteria{From: from, To: to.Add(-time.Second), ProjectIDs: []int{1}, Limit: 2})
		require.NoError(t, err)
		require.Len(t, tags, 2)
		tagValues, err := store.ListTagValues(ctx, &warnly.ListTagValuesCriteria{From: from, To: to.Add(-time.Second), ProjectIDs: []int{1}, Tag: fixtureEnv, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []warnly.TagValueCount{{Value: fixtureProd, Count: 1}, {Value: "stage", Count: 1}}, tagValues)
		tagValues, err = store.ListTagValues(ctx, &warnly.ListTagValuesCriteria{From: from, To: to.Add(-time.Second), ProjectIDs: []int{1}, Tag: fixtureSpecial, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []warnly.TagValueCount{{Value: fixtureQuote, Count: 2}}, tagValues)
	})

	t.Run("inclusive and exclusive upper bounds", func(t *testing.T) {
		criteria := *ec
		criteria.From = to
		count, err := store.CountEvents(ctx, &criteria)
		require.NoError(t, err)
		require.Zero(t, count)
		metrics, err := store.ListIssueMetrics(ctx, &warnly.ListIssueMetricsCriteria{From: to, To: to, ProjectIDs: []int{1}, GroupIDs: []int64{10}})
		require.NoError(t, err)
		require.Len(t, metrics, 1)
		require.EqualValues(t, 1, metrics[0].TimesSeen)
	})

	t.Run("replace tags and preserve transaction on failure", func(t *testing.T) {
		ev := event(20)
		ev.ProjectID = 3
		require.NoError(t, store.StoreEvent(ctx, ev))
		ev.TagsKey, ev.TagsValue = []string{"new"}, []string{"tag"}
		require.NoError(t, store.StoreEvent(ctx, ev))
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM event_tag WHERE pid = 3").Scan(&count))
		require.Equal(t, 1, count)
		// Fail after the previous row and tags have been deleted within the transaction.
		// DATETIME rejects years beyond 9999, including the calculated expiry.
		ev.CreatedAt = time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)
		require.Error(t, store.StoreEvent(ctx, ev))
		got, err := store.GetIssueEvent(ctx, &warnly.EventDefCriteria{ProjectID: 3, GroupID: 10, EventID: ev.EventID})
		require.NoError(t, err)
		require.Equal(t, []string{"new"}, got.TagsKey)
		require.Equal(t, []string{"tag"}, got.TagsValue)
	})

	t.Run("top four values per tag and UTC", func(t *testing.T) {
		for i := range 6 {
			ev := event(30 + i)
			ev.ProjectID = 4
			ev.CreatedAt = now.In(time.FixedZone("UTC+3", 3*60*60))
			ev.TagsKey, ev.TagsValue = []string{"value"}, []string{strconv.Itoa(i)}
			require.NoError(t, store.StoreEvent(ctx, ev))
		}
		values, err := store.CountFields(ctx, &warnly.EventDefCriteria{From: from, To: to, ProjectID: 4, GroupID: 10})
		require.NoError(t, err)
		require.Len(t, values, 4)
		hours, err := store.CalculateEvents(ctx, &warnly.ListIssueMetricsCriteria{From: from, To: to, ProjectIDs: []int{4}})
		require.NoError(t, err)
		require.Equal(t, []warnly.EventsPerHour{{TS: now, ProjectID: 4, Count: 6}}, hours)
	})

	t.Run("pagination", func(t *testing.T) {
		p, err := store.GetEventPagination(ctx, &warnly.EventPaginationCriteria{From: from, To: to.Add(-time.Second), ProjectID: 1, GroupID: 10, EventID: b.EventID, CreatedAt: b.CreatedAt})
		require.NoError(t, err)
		require.Equal(t, &warnly.EventPagination{FirstEventID: c.EventID, LastEventID: a.EventID, NextEventID: a.EventID, PrevEventID: c.EventID}, p)
	})

	t.Run("system queries", func(t *testing.T) {
		schemas, err := store.ListSchemas(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(schemas), 2)
		_, err = store.ListErrors(ctx, warnly.ListErrorsCriteria{LastErrorTime: from})
		require.NoError(t, err)
		_, err = store.ListSlowQueries(ctx)
		require.NoError(t, err)
	})

	t.Run("retention and cancellation", func(t *testing.T) {
		criteria := *ec
		criteria.From = now.AddDate(0, 0, -3)
		count, err := store.CountEvents(ctx, &criteria)
		require.NoError(t, err)
		require.EqualValues(t, 3, count)
		require.NoError(t, store.PurgeExpired(ctx, time.Now()))
		for _, table := range []string{"event", "event_tag"} {
			var n int
			require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE event_id = ?", expired.EventID).Scan(&n))
			require.Zero(t, n)
		}
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, err = store.CountEvents(cancelled, ec)
		require.ErrorIs(t, err, context.Canceled)
	})
}
