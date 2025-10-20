package ch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/vk-rv/warnly/internal/svcotel"
	"github.com/vk-rv/warnly/internal/warnly"
	"go.opentelemetry.io/otel/trace"
)

const (
	hasTagSQL    = " AND has(_tags_hash_map, cityHash64(?))"
	notHasTagSQL = " AND not has(_tags_hash_map, cityHash64(?))"
)

// ClickhouseStore encapsulates clickhouse connection.
type ClickhouseStore struct {
	conn            clickhouse.Conn
	tracer          trace.Tracer // https://github.com/ClickHouse/clickhouse-go/issues/1444
	asyncInsertWait bool
}

// NewClickhouseStore creates a new ClickhouseStore.
func NewClickhouseStore(conn clickhouse.Conn, tracerProvider svcotel.TracerProvider) *ClickhouseStore {
	return &ClickhouseStore{conn: conn, tracer: tracerProvider.Tracer("clickhouse")}
}

func (c *ClickhouseStore) EnableAsyncInsertWait() {
	c.asyncInsertWait = true
}

// Close closes the connection to Clickhouse.
func (s *ClickhouseStore) Close() error {
	return s.conn.Close()
}

// ListIssueMetrics lists issue metrics for the given project IDs and issue IDs within the specified time range.
// It displays how many times each issue was seen, when it was first and last seen,
// and the number of unique users affected.
func (s *ClickhouseStore) ListIssueMetrics(
	ctx context.Context,
	c *warnly.ListIssueMetricsCriteria,
) ([]warnly.IssueMetrics, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListIssueMetrics")
	defer span.End()

	gidQuestionMarks, args := createPlaceholdersAndArgs(c.GroupIDs)
	pidQuestionMarks, pidArgs := createPlaceholdersAndArgs(c.ProjectIDs)

	args = append(args, c.From, c.To)
	args = append(args, pidArgs...)

	query := `SELECT gid, 
					 count() AS times_seen, 
					 min(created_at) AS first_seen, 
					 max(created_at) AS last_seen,
					 ifNull(uniq(nullIf(user, '')), 0) AS user_count 
			   FROM event 
			   WHERE deleted = 0
			   AND gid IN (` + strings.Join(gidQuestionMarks, ",") + `)
			   AND created_at >= toDateTime(?, 'UTC')
			   AND created_at <= toDateTime(?, 'UTC')
			   AND pid IN (` + strings.Join(pidQuestionMarks, ",") + `)
			   GROUP BY gid`

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list issue metrics: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	var res []warnly.IssueMetrics
	for rows.Next() {
		m := warnly.IssueMetrics{}
		if err := rows.Scan(&m.GID, &m.TimesSeen, &m.FirstSeen, &m.LastSeen, &m.UserCount); err != nil {
			return nil, fmt.Errorf("clickhouse: list issue metrics, scan result: %w", err)
		}
		res = append(res, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list issue metrics, rows.Err: %w", err)
	}

	return res, nil
}

// CountFields counts additional fields for a given issue and project within a specified time range.
func (s *ClickhouseStore) CountFields(ctx context.Context, c *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.CountFields")
	defer span.End()

	const query = `SELECT 
				   	tag,
				   	value,
				   	count() AS count,
				   	min(created_at) AS first_seen,
				   	max(created_at) AS last_seen
				   FROM event
				   ARRAY JOIN 
				   	tags.key AS tag,
				   	tags.value AS value
				   WHERE gid = ?
				   AND deleted = 0
				   AND created_at >= toDateTime(?, 'UTC')
				   AND created_at < toDateTime(?, 'UTC')
				   AND pid = ?
				   GROUP BY tag, value
				   ORDER BY count DESC
				   LIMIT 4 BY tag 
				   LIMIT 1000`

	rows, err := s.conn.Query(ctx, query, c.GroupID, c.From, c.To, c.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: count tags: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res := make([]warnly.FieldValueNum, 0, 10)
	for rows.Next() {
		tv := warnly.FieldValueNum{}
		if err := rows.Scan(&tv.Tag, &tv.Value, &tv.Count, &tv.FirstSeen, &tv.LastSeen); err != nil {
			return nil, fmt.Errorf("clickhouse: count tags, scan result: %w", err)
		}
		res = append(res, tv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: count tags, rows.Err: %w", err)
	}

	return res, nil
}

// StoreEvent stores an event in the analytics database.
func (s *ClickhouseStore) StoreEvent(ctx context.Context, ev *warnly.EventClickhouse) error {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.StoreEvent")
	defer span.End()

	const query = `INSERT INTO event (
		created_at, sdk_version, user, primary_hash, env, event_id,
		message, ipv6, release, title, ipv4,
		exception_frames.in_app, contexts.key, exception_frames.colno, exception_frames.abs_path,
		exception_frames.lineno, exception_stacks.type, exception_stacks.value, tags.key,
		exception_frames.function, tags.value, exception_frames.filename, contexts.value,
		gid, user_name, user_username, user_email, pid, level, type, sdk_id, platform, retention_days, deleted
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := s.conn.AsyncInsert(
		ctx,
		query,
		s.asyncInsertWait, // No need to wait for acknowledgment for async insert depends on testing
		ev.CreatedAt,
		ev.SDKVersion,
		ev.User,
		ev.PrimaryHash,
		ev.Env,
		ev.EventID,
		ev.Message,
		ev.IPv6,
		ev.Release,
		ev.Title,
		ev.IPv4,
		ev.ExceptionFramesInApp,
		ev.ContextsKey,
		ev.ExceptionFramesColNo,
		ev.ExceptionFramesAbsPath,
		ev.ExceptionFramesLineNo,
		ev.ExceptionStacksType,
		ev.ExceptionStacksValue,
		ev.TagsKey,
		ev.ExceptionFramesFunction,
		ev.TagsValue,
		ev.ExceptionFramesFilename,
		ev.ContextsValue,
		ev.GroupID,
		ev.UserName,
		ev.UserUsername,
		ev.UserEmail,
		ev.ProjectID,
		ev.Level,
		ev.Type,
		ev.SDKID,
		ev.Platform,
		ev.RetentionDays,
		ev.Deleted,
	); err != nil {
		return fmt.Errorf("clickhouse: async insert event: %w", err)
	}

	return nil
}

// GetIssueEvent retrieves a single event associated with a specific issue and project within a given time range.
func (s *ClickhouseStore) GetIssueEvent(ctx context.Context, c *warnly.EventDefCriteria) (*warnly.IssueEvent, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.GetIssueEvent")
	defer span.End()

	var (
		query string
		args  []any
	)

	if c.EventID != "" {
		query = `SELECT replaceAll(toString(event_id), '-', '') AS event_id,
			env, release,
			user, user_username, user_name, user_email,
			tags.key, tags.value, message,
			exception_frames.abs_path, exception_frames.colno,
			exception_frames.function, exception_frames.lineno,
			exception_frames.in_app
			FROM event WHERE
			deleted = 0
			AND event_id = ?
			AND pid = ?
			AND gid = ?
			LIMIT 1`
		args = []any{c.EventID, c.ProjectID, c.GroupID}
	} else {
		query = `SELECT replaceAll(toString(event_id), '-', '') AS event_id,
			env, release,
			user, user_username, user_name, user_email,
			tags.key, tags.value, message,
			exception_frames.abs_path, exception_frames.colno,
			exception_frames.function, exception_frames.lineno,
			exception_frames.in_app
			FROM event WHERE
			deleted = 0
			AND created_at >= toDateTime(?, 'UTC')
			AND created_at < toDateTime(?, 'UTC')
			AND pid = ?
			AND gid = ?
			LIMIT 1`
		args = []any{c.From, c.To, c.ProjectID, c.GroupID}
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: get issue: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	i := warnly.IssueEvent{}
	if rows.Next() {
		if err := rows.Scan(
			&i.EventID,
			&i.Env,
			&i.Release,
			&i.UserID,
			&i.UserUsername,
			&i.UserName,
			&i.UserEmail,
			&i.TagsKey,
			&i.TagsValue,
			&i.Message,
			&i.ExceptionFramesAbsPath,
			&i.ExceptionFramesColno,
			&i.ExceptionFramesFunction,
			&i.ExceptionFramesLineno,
			&i.ExceptionFramesInApp); err != nil {
			return nil, fmt.Errorf("clickhouse: get issue, scan result: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: get issue, rows.Err: %w", err)
	}

	return &i, nil
}

// ListFieldFilters lists field filters for a given set of project IDs.
func (s *ClickhouseStore) ListFieldFilters(
	ctx context.Context,
	criteria *warnly.FieldFilterCriteria,
) ([]warnly.Filter, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListFieldFilters")
	defer span.End()

	pidPlaceholders, pidArgs := createPlaceholdersAndArgs(criteria.ProjectIDs)

	args := make([]any, 0, len(pidArgs)+2)
	args = append(args, pidArgs...)
	args = append(args, criteria.From, criteria.To)

	query := `SELECT
				tags.key AS tag_key,
				tags.value AS tag_value,
				count() AS frequency
			FROM event
			ARRAY JOIN tags
			WHERE
				deleted = 0
			AND pid IN (` + strings.Join(pidPlaceholders, ",") + `)
			AND created_at >= toDateTime(?, 'UTC')
			AND created_at < toDateTime(?, 'UTC')
			GROUP BY tag_key, tag_value
			ORDER BY frequency DESC
			LIMIT 1000`

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list field filters: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var (
		filters   []warnly.Filter
		frequency uint64
	)
	for rows.Next() {
		var f warnly.Filter
		if err := rows.Scan(&f.Key, &f.Value, &frequency); err != nil {
			return nil, fmt.Errorf("clickhouse: list field filters, scan result: %w", err)
		}
		filters = append(filters, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list field filters, rows.Err: %w", err)
	}

	return filters, nil
}

// CalculateFields calculates the number of occurrences of each field for a given issue identifier and
// project within a specified time range.
func (s *ClickhouseStore) CalculateFields(ctx context.Context, c warnly.FieldsCriteria) ([]warnly.TagCount, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.CalculateFields")
	defer span.End()

	const query = `
		SELECT 
    		arrayJoin(tags.key) AS tag,
    		count() AS count
		FROM event
		PREWHERE 
    		gid = ?
    		AND deleted = 0
		WHERE 
    		created_at >= toDateTime(?, 'UTC')
    	AND created_at < toDateTime(?, 'UTC')
    	AND pid = ?
		GROUP BY tag
		ORDER BY count DESC
		LIMIT 1000`

	rows, err := s.conn.Query(ctx, query, c.IssueID, c.From, c.To, c.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: calculate tags: %w", err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res := make([]warnly.TagCount, 0, 10)
	for rows.Next() {
		tc := warnly.TagCount{}
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, fmt.Errorf("clickhouse: calculate tags, scan result: %w", err)
		}
		res = append(res, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: calculate tags, rows.Err: %w", err)
	}

	return res, nil
}

// CalculateEventsPerDay calculates the number of events per day for a given group and project
// within a specified time range.
func (s *ClickhouseStore) CalculateEventsPerDay(ctx context.Context, c *warnly.EventDefCriteria) ([]warnly.EventPerDay, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.CalculateEventsPerDay")
	defer span.End()

	const query = `SELECT 
				   	gid,
					toDate(created_at, 'UTC') AS time,
					count() AS event_count
				   FROM event
				   WHERE deleted = 0
				   AND gid = ?
				   AND pid = ?
				   AND created_at >= toDateTime(?, 'UTC')
				   AND created_at < toDateTime(?, 'UTC')
				   GROUP BY gid, time
				   ORDER BY time DESC, gid ASC`

	rows, err := s.conn.Query(ctx, query, c.GroupID, c.ProjectID, c.From, c.To)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: calculate events per day: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res := make([]warnly.EventPerDay, 0, 31)
	for rows.Next() {
		e := warnly.EventPerDay{}
		if err := rows.Scan(&e.GID, &e.Time, &e.Count); err != nil {
			return nil, fmt.Errorf("clickhouse: calculate events per day, scan result: %w", err)
		}
		res = append(res, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: calculate events per day, rows.Err: %w", err)
	}

	return res, nil
}

// ListEvents lists error events per issue based on the given criteria.
func (s *ClickhouseStore) ListEvents(ctx context.Context, criteria *warnly.EventCriteria) ([]warnly.EventEntry, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListEvents")
	defer span.End()

	query := `SELECT 
			  	replaceAll(toString(event_id), '-', '') AS event_id,
				created_at,
				title,
				message,
				release,
				env,
				user,
				user_email,
				user_username,
				user_name,
				tags.value[indexOf(tags.key, 'os')] AS os
			FROM event
			WHERE deleted = 0
			AND gid = ?
			AND created_at >= toDateTime(?, 'UTC')
			AND created_at < toDateTime(?, 'UTC')`

	args := []any{criteria.GroupID, criteria.From, criteria.To}

	if criteria.Message != "" {
		query += " AND notEquals(positionCaseInsensitive(message, ?), 0)"
		args = append(args, criteria.Message)
	}

	for key, value := range criteria.Tags {
		if value.IsNot {
			query += notHasTagSQL
		} else {
			query += hasTagSQL
		}
		args = append(args, fmt.Sprintf("%s=%s", key, value.Value))
	}

	query += " AND in(pid, ?)"
	args = append(args, criteria.ProjectID)

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, criteria.Limit, criteria.Offset)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list events: %w", err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var events []warnly.EventEntry
	for rows.Next() {
		var event warnly.EventEntry
		if err := rows.Scan(
			&event.EventID,
			&event.CreatedAt,
			&event.Title,
			&event.Message,
			&event.Release,
			&event.Env,
			&event.User,
			&event.UserEmail,
			&event.UserUsername,
			&event.UserName,
			&event.OS); err != nil {
			return nil, fmt.Errorf("clickhouse: list events, scan result: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list events, rows.Err: %w", err)
	}

	return events, nil
}

// CountEvents returns the number of events for a given project and issue.
func (s *ClickhouseStore) CountEvents(ctx context.Context, criteria *warnly.EventCriteria) (uint64, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.CountEvents")
	defer span.End()

	query := `SELECT count() AS count
			  FROM event 
			  PREWHERE gid = ?
			  WHERE deleted = 0
			  AND created_at >= toDateTime(?, 'UTC')
			  AND created_at < toDateTime(?, 'UTC')`

	args := []any{criteria.GroupID, criteria.From, criteria.To}

	if criteria.Message != "" {
		query += " AND notEquals(positionCaseInsensitive(message, ?), 0)"
		args = append(args, criteria.Message)
	}

	for key, value := range criteria.Tags {
		if value.IsNot {
			query += " AND not has(_tags_hash_map, cityHash64(?))"
		} else {
			query += " AND has(_tags_hash_map, cityHash64(?))"
		}
		args = append(args, fmt.Sprintf("%s=%s", key, value.Value))
	}

	query += " AND in(pid, ?)"
	args = append(args, criteria.ProjectID)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("clickhouse: count events: %w", err)
	}

	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	var count uint64
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, fmt.Errorf("clickhouse: count events, scan result: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("clickhouse: count events, rows.Err: %w", err)
	}

	return count, nil
}

// ListSchemas lists olap database schemas from largest to smallest.
func (s *ClickhouseStore) ListSchemas(ctx context.Context) ([]warnly.Schema, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListSchemas")
	defer span.End()

	const query = `SELECT name, 
						  formatReadableSize(total_bytes) AS readable_bytes,
						  total_bytes, 
						  total_rows, 
						  engine, 
						  partition_key
				   FROM system.tables ORDER BY total_bytes DESC`

	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: get schema: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	schemas := make([]warnly.Schema, 0, 10)
	for rows.Next() {
		var schema warnly.Schema
		err := rows.Scan(
			&schema.Name,
			&schema.ReadableBytes,
			&schema.TotalBytes,
			&schema.TotalRows,
			&schema.Engine,
			&schema.PartitionKey)
		if err != nil {
			return nil, fmt.Errorf("clickhouse: get schema, scan result: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: get schema, rows.Err: %w", err)
	}

	return schemas, nil
}

// ListErrors lists recent errors from the olap system for the last 24 hours.
func (s *ClickhouseStore) ListErrors(ctx context.Context, c warnly.ListErrorsCriteria) ([]warnly.AnalyticsStoreErr, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListErrors")
	defer span.End()

	const query = `SELECT name, count() AS count, max(last_error_time) AS max_last_error_time
				   FROM system.errors
				   WHERE last_error_time > toDateTime(?)
				   GROUP BY name
				   ORDER BY count DESC`

	rows, err := s.conn.Query(ctx, query, c.LastErrorTime)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list errors: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	errors := make([]warnly.AnalyticsStoreErr, 0, 10)
	for rows.Next() {
		var e warnly.AnalyticsStoreErr
		if err := rows.Scan(&e.Name, &e.Count, &e.MaxLastErrorTime); err != nil {
			return nil, fmt.Errorf("clickhouse: list errors, scan result: %w", err)
		}
		errors = append(errors, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list errors, rows.Err: %w", err)
	}

	return errors, nil
}

// ListSlowQueries lists slow SQL queries from the olap system and their statistics.
func (s *ClickhouseStore) ListSlowQueries(ctx context.Context) ([]warnly.SQLQuery, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListSlowQueries")
	defer span.End()

	const query = `SELECT 
    				normalizeQuery(query) AS normalized_query,
    				avg(query_duration_ms) AS avg_duration,
    				avg(result_rows) AS avg_result_rows,
    				count() / greatest(1, dateDiff('minute', min(event_time), max(event_time))) AS calls_per_minute,
    				count() AS total_calls,
    				sum(read_bytes) AS rb,
    				formatReadableSize(sum(read_bytes)) AS total_read_bytes,
    				sum(read_bytes) / sum(sum(read_bytes)) OVER () * 100 AS percentage_iops,
    				sum(query_duration_ms) / sum(sum(query_duration_ms)) OVER () * 100 AS percentage_runtime,
    				toString(normalized_query_hash) AS normalized_query_hash
				FROM system.query_log
				WHERE is_initial_query = 1
				AND event_time >= now() - INTERVAL 30 DAY
				AND type = 2
				GROUP BY normalized_query_hash, normalized_query
				ORDER BY sum(read_bytes) DESC
				LIMIT 10`

	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list slow queries: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	queries := make([]warnly.SQLQuery, 0, 10)
	for rows.Next() {
		var q warnly.SQLQuery
		if err := rows.Scan(
			&q.NormalizedQuery,
			&q.AvgDuration,
			&q.AvgResultRows,
			&q.CallsPerMinute,
			&q.TotalCalls,
			&q.ReadBytes,
			&q.TotalReadBytes,
			&q.PercentageIOPS,
			&q.PercentageRuntime,
			&q.NormalizedQueryHash); err != nil {
			return nil, fmt.Errorf("clickhouse: list slow queries, scan result: %w", err)
		}
		queries = append(queries, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list slow queries, rows.Err: %w", err)
	}

	var totalReadBytes uint64
	for _, q := range queries {
		totalReadBytes += q.ReadBytes
	}
	for i := range queries {
		queries[i].TotalReadBytesNumeric = float64(totalReadBytes)
		queries[i].Percent = float64(queries[i].ReadBytes) / float64(totalReadBytes) * 100
	}

	return queries, nil
}

// CalculateEvents calculates the number of events per day split by hour.
//
//nolint:staticcheck // false positive
func (s *ClickhouseStore) CalculateEvents(
	ctx context.Context,
	c *warnly.ListIssueMetricsCriteria,
) ([]warnly.EventsPerHour, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.CalculateEvents")
	defer span.End()

	pidQuestionMarks := make([]string, 0, len(c.ProjectIDs))
	for range c.ProjectIDs {
		pidQuestionMarks = append(pidQuestionMarks, "?")
	}

	pidArgs := make([]any, 0, len(c.ProjectIDs))
	for _, pid := range c.ProjectIDs {
		pidArgs = append(pidArgs, pid)
	}

	args := make([]any, 0, len(pidArgs)+2)
	args = append(args, pidArgs...)
	args = append(args, c.From, c.To)

	query := `SELECT 
    		  	toStartOfHour(created_at, 'UTC') AS ts,
				pid,
				count() AS event_count
			  FROM event
			  WHERE deleted = 0
			  AND pid IN (` + strings.Join(pidQuestionMarks, ",") + `)
			  AND created_at >= toDateTime(?, 'UTC')
			  AND created_at < toDateTime(?, 'UTC')
			  GROUP BY ts, pid
			  ORDER BY ts ASC
			  LIMIT 5000`

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: calculate events: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	res := make([]warnly.EventsPerHour, 0, 24)
	for rows.Next() {
		var (
			ts    time.Time
			pid   uint16
			count uint64
		)
		if err := rows.Scan(&ts, &pid, &count); err != nil {
			return nil, fmt.Errorf("clickhouse: calculate events, scan result: %w", err)
		}
		res = append(res, warnly.EventsPerHour{TS: ts, ProjectID: int(pid), Count: int(count)})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: calculate events, rows.Err: %w", err)
	}

	return res, nil
}

// ListPopularTags lists popular tag keys across all events in the given time range and projects.
func (s *ClickhouseStore) ListPopularTags(
	ctx context.Context,
	c *warnly.ListPopularTagsCriteria,
) ([]warnly.TagCount, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListPopularTags")
	defer span.End()

	pidQuestionMarks, pidArgs := createPlaceholdersAndArgs(c.ProjectIDs)

	args := []any{c.From, c.To}
	args = append(args, pidArgs...)

	query := `SELECT tag, count() AS count
			   FROM event
			   ARRAY JOIN tags.key AS tag
			   WHERE deleted = 0
			   AND created_at >= toDateTime(?, 'UTC')
			   AND created_at <= toDateTime(?, 'UTC')
			   AND pid IN (` + strings.Join(pidQuestionMarks, ",") + `)
			   GROUP BY tag
			   ORDER BY count DESC
			   LIMIT ?`

	args = append(args, c.Limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list popular tags: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res := make([]warnly.TagCount, 0, c.Limit)
	for rows.Next() {
		tc := warnly.TagCount{}
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, fmt.Errorf("clickhouse: list popular tags, scan result: %w", err)
		}
		res = append(res, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list popular tags, rows.Err: %w", err)
	}

	return res, nil
}

// ListTagValues lists popular values for a given tag in the specified time range and projects.
func (s *ClickhouseStore) ListTagValues(
	ctx context.Context,
	c *warnly.ListTagValuesCriteria,
) ([]warnly.TagValueCount, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.ListTagValues")
	defer span.End()

	pidQuestionMarks, pidArgs := createPlaceholdersAndArgs(c.ProjectIDs)

	args := []any{c.Tag, c.From, c.To}
	args = append(args, pidArgs...)

	query := `SELECT value, count() AS count
			   FROM event
			   ARRAY JOIN tags.key AS tag, tags.value AS value
			   WHERE tag = ?
			   AND deleted = 0
			   AND created_at >= toDateTime(?, 'UTC')
			   AND created_at <= toDateTime(?, 'UTC')
			   AND pid IN (` + strings.Join(pidQuestionMarks, ",") + `)
			   GROUP BY value
			   ORDER BY count DESC
			   LIMIT ?`

	args = append(args, c.Limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: list tag values: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res := make([]warnly.TagValueCount, 0, c.Limit)
	for rows.Next() {
		tvc := warnly.TagValueCount{}
		if err := rows.Scan(&tvc.Value, &tvc.Count); err != nil {
			return nil, fmt.Errorf("clickhouse: list tag values, scan result: %w", err)
		}
		res = append(res, tvc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: list tag values, rows.Err: %w", err)
	}

	return res, nil
}

// GetFilteredGroupIDs returns group IDs that match the query filters.
func (s *ClickhouseStore) GetFilteredGroupIDs(
	ctx context.Context,
	tokens []warnly.QueryToken,
	from, to time.Time,
	projectIDs []int,
) ([]int64, error) {
	ctx, span := s.tracer.Start(ctx, "ClickhouseStore.GetFilteredGroupIDs")
	defer span.End()

	query := `SELECT DISTINCT gid FROM event WHERE deleted = 0 AND pid IN (?` +
		strings.Repeat(",?", len(projectIDs)-1) +
		`) AND created_at >= toDateTime(?, 'UTC') AND created_at <= toDateTime(?, 'UTC')`

	args := make([]any, 0, len(projectIDs)+2)
	for _, pid := range projectIDs {
		args = append(args, pid)
	}
	args = append(args, from, to)

	for _, token := range tokens {
		if token.IsRawText {
			query += " AND (notEquals(positionCaseInsensitive(message, ?), 0) OR notEquals(positionCaseInsensitive(title, ?), 0))"
			args = append(args, token.Value, token.Value)
		} else {
			hash := fmt.Sprintf("%s=%s", token.Key, token.Value)
			if token.Operator == "is not" {
				query += " AND not has(_tags_hash_map, cityHash64(?))"
			} else {
				query += " AND has(_tags_hash_map, cityHash64(?))"
			}
			args = append(args, hash)
		}
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: get filtered group ids: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	var gids []int64
	for rows.Next() {
		var gid uint64
		if err := rows.Scan(&gid); err != nil {
			return nil, fmt.Errorf("clickhouse: get filtered group ids, scan result: %w", err)
		}
		gids = append(gids, int64(gid))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse: get filtered group ids, rows.Err: %w", err)
	}

	return gids, nil
}

// createPlaceholdersAndArgs creates SQL placeholders and corresponding args for the given items.
func createPlaceholdersAndArgs[T any](items []T) ([]string, []any) {
	placeholders := make([]string, len(items))
	args := make([]any, len(items))
	for i := range items {
		placeholders[i] = "?"
		args[i] = items[i]
	}
	return placeholders, args
}
