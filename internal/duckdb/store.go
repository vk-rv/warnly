package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vk-rv/warnly/internal/warnly"
)

var _ warnly.AnalyticsStore = (*Store)(nil)

var tables = [...]string{"event_tag", "event"}

// Store keeps events and their searchable tags in DuckDB tables in one MySQL schema.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// StoreEvent atomically replaces an event and its tags, making ingestion retries idempotent.
// The full payload preserves ordered contexts and stack frames without lossy SQL conversions.
func (s *Store) StoreEvent(ctx context.Context, event *warnly.EventClickhouse) error {
	if event == nil {
		return errors.New("mysql-duckdb: nil event")
	}

	if len(event.TagsKey) != len(event.TagsValue) {
		return errors.New("mysql-duckdb: tag keys and values have different lengths")
	}

	id, err := canonicalID(event.EventID)
	if err != nil {
		return err
	}

	ev := *event
	ev.EventID = id
	ev.CreatedAt = ev.CreatedAt.UTC().Truncate(time.Microsecond)
	payload, err := json.Marshal(&ev)
	if err != nil {
		return fmt.Errorf("mysql-duckdb: marshal event: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // committed transactions are already closed

	for i := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+tables[i]+" WHERE pid = ? AND event_id = ?", ev.ProjectID, id); err != nil {
			return fmt.Errorf("mysql-duckdb: replace %s: %w", tables[i], err)
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO event
		(pid, event_id, gid, created_at, event_day, event_hour, expires_at, deleted, user_id, message, title, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ProjectID,
		id,
		ev.GroupID,
		ev.CreatedAt,
		ev.CreatedAt.Format(time.DateOnly),
		ev.CreatedAt.Truncate(time.Hour),
		ev.CreatedAt.AddDate(0, 0, int(ev.RetentionDays)),
		ev.Deleted,
		ev.User,
		ev.Message,
		ev.Title,
		string(payload))
	if err != nil {
		return fmt.Errorf("mysql-duckdb: insert event: %w", err)
	}
	for i, key := range ev.TagsKey {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO event_tag (pid, event_id, ordinal, tag_key, tag_value)
			VALUES (?, ?, ?, ?, ?)`,
			ev.ProjectID,
			id,
			i,
			key,
			ev.TagsValue[i])
		if err != nil {
			return fmt.Errorf("mysql-duckdb: insert tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysql-duckdb: store event, commit transaction: %w", err)
	}

	return nil
}

// canonicalID converts a UUID string to a 32-character hex string without dashes.
func canonicalID(value string) (string, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf("mysql-duckdb: invalid event ID: %w", err)
	}
	return strings.ReplaceAll(id.String(), "-", ""), nil
}

// PurgeExpired implements physical retention; reads exclude expired events immediately.
func (s *Store) PurgeExpired(ctx context.Context, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // committed transactions are already closed
	_, err = tx.ExecContext(
		ctx,
		`DELETE FROM event_tag WHERE EXISTS
		(SELECT 1 FROM event e WHERE e.pid = event_tag.pid AND e.event_id = event_tag.event_id AND e.expires_at <= ?)`,
		now.UTC())
	if err != nil {
		return fmt.Errorf("mysql-duckdb: purge tags: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM event WHERE expires_at <= ?", now.UTC()); err != nil {
		return fmt.Errorf("mysql-duckdb: purge events: %w", err)
	}

	return tx.Commit()
}

type predicate struct {
	sql  string
	args []any
}

func active() predicate {
	return predicate{sql: "e.deleted = 0 AND e.expires_at > ?", args: []any{time.Now().UTC()}}
}

func (p *predicate) add(query string, args ...any) {
	p.sql += " AND " + query
	p.args = append(p.args, args...)
}

func (p *predicate) period(from, to time.Time, inclusive bool) {
	op := "<"
	if inclusive {
		op = "<="
	}
	p.add("e.created_at >= ? AND e.created_at "+op+" ?", from.UTC(), to.UTC())
}

func in[T ~int | ~int64](p *predicate, column string, values []T) {
	if len(values) == 0 {
		p.add("1 = 0")
		return
	}
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	p.add(column+" IN (?"+strings.Repeat(",?", len(values)-1)+")", args...)
}

func (p *predicate) tag(key, value string, negate bool) {
	op := "EXISTS"
	if negate {
		op = "NOT EXISTS"
	}
	p.add(op+` (SELECT 1 FROM event_tag f WHERE f.pid = e.pid AND f.event_id = e.event_id
		AND CAST(f.tag_key AS CHAR) = ? AND CAST(f.tag_value AS CHAR) = ?)`, key, value)
}

func eventPredicate(c *warnly.EventCriteria) predicate {
	p := active()

	p.add("e.pid = ? AND e.gid = ?", c.ProjectID, c.GroupID)
	p.period(c.From, c.To, false)
	if c.Message != "" {
		p.add("LOCATE(LOWER(?), LOWER(e.message)) > 0", c.Message)
	}
	for key, value := range c.Tags {
		p.tag(key, value.Value, value.IsNot)
	}

	return p
}

func defPredicate(c *warnly.EventDefCriteria) predicate {
	p := active()
	p.add("e.pid = ? AND e.gid = ?", c.ProjectID, c.GroupID)
	p.period(c.From, c.To, false)
	return p
}

const tagJoin = " FROM event e JOIN event_tag t ON t.pid = e.pid AND t.event_id = e.event_id WHERE "

// collect propagates query, scan and iteration errors for all analytics queries.
func collect[T any](ctx context.Context, db *sql.DB, query string, args []any, scan func(*sql.Rows, *T) error) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql-duckdb: query: %w", err)
	}
	defer rows.Close()

	result := make([]T, 0)

	for rows.Next() {
		var item T
		if err := scan(rows, &item); err != nil {
			return nil, fmt.Errorf("mysql-duckdb: scan: %w", err)
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

func (s *Store) CountEvents(ctx context.Context, c *warnly.EventCriteria) (uint64, error) {
	p := eventPredicate(c)
	var count uint64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM event e WHERE "+p.sql, p.args...).Scan(&count)
	return count, err
}

func (s *Store) ListEvents(ctx context.Context, c *warnly.EventCriteria) ([]warnly.EventEntry, error) {
	p := eventPredicate(c)
	p.args = append(p.args, c.Limit, c.Offset)
	query := "SELECT e.payload FROM event e WHERE " + p.sql +
		" ORDER BY e.created_at DESC, e.event_id DESC LIMIT ? OFFSET ?"
	if c.Message != "" || len(c.Tags) > 0 {
		query = "SELECT payload FROM (SELECT e.payload, e.created_at, e.event_id FROM event e WHERE " + p.sql +
			" LIMIT 18446744073709551615) matched ORDER BY created_at DESC, event_id DESC LIMIT ? OFFSET ?"
	}
	return collect(ctx, s.db, query, p.args,
		func(rows *sql.Rows, entry *warnly.EventEntry) error {
			var payload []byte
			if err := rows.Scan(&payload); err != nil {
				return err
			}
			var ev warnly.EventClickhouse
			if err := json.Unmarshal(payload, &ev); err != nil {
				return err
			}
			*entry = warnly.EventEntry{
				EventID: ev.EventID, CreatedAt: ev.CreatedAt, Title: ev.Title,
				Message: ev.Message, Release: ev.Release, Env: ev.Env, User: ev.User, UserEmail: ev.UserEmail,
				UserUsername: ev.UserUsername, UserName: ev.UserName,
			}
			for i, key := range ev.TagsKey {
				if key == "os" {
					entry.OS = ev.TagsValue[i]
					break
				}
			}
			return nil
		})
}

func (s *Store) GetIssueEvent(ctx context.Context, c *warnly.EventDefCriteria) (*warnly.IssueEvent, error) {
	p := active()
	p.add("e.pid = ? AND e.gid = ?", c.ProjectID, c.GroupID)
	if c.EventID != "" {
		id, err := canonicalID(c.EventID)
		if err != nil {
			return nil, err
		}
		p.add("e.event_id = ?", id)
	} else {
		p.period(c.From, c.To, false)
	}
	var payload []byte
	err := s.db.QueryRowContext(ctx, "SELECT e.payload FROM event e WHERE "+p.sql+
		" ORDER BY e.created_at DESC, e.event_id DESC LIMIT 1", p.args...).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return &warnly.IssueEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	var ev warnly.EventClickhouse
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}

	return &warnly.IssueEvent{
		EventID: ev.EventID, CreatedAt: ev.CreatedAt, Env: ev.Env, Release: ev.Release,
		UserID: ev.User, UserEmail: ev.UserEmail, UserName: ev.UserName, UserUsername: ev.UserUsername,
		TagsKey: ev.TagsKey, TagsValue: ev.TagsValue, ContextsKey: ev.ContextsKey, ContextsValue: ev.ContextsValue,
		Message: ev.Message, ExceptionFramesAbsPath: ev.ExceptionFramesAbsPath,
		ExceptionFramesFunction: ev.ExceptionFramesFunction, ExceptionFramesColno: ints(ev.ExceptionFramesColNo),
		ExceptionFramesLineno: ints(ev.ExceptionFramesLineNo), ExceptionFramesInApp: ints(ev.ExceptionFramesInApp),
	}, nil
}

func ints[T ~uint8 | ~uint32](values []T) []int {
	result := make([]int, len(values))
	for i, value := range values {
		result[i] = int(value)
	}
	return result
}

func (s *Store) ListIssueMetrics(ctx context.Context, c *warnly.ListIssueMetricsCriteria) ([]warnly.IssueMetrics, error) {
	p := active()
	p.period(c.From, c.To, true)
	in(&p, "e.pid", c.ProjectIDs)
	in(&p, "e.gid", c.GroupIDs)
	return collect(ctx, s.db, `SELECT e.gid, COUNT(*), MIN(e.created_at), MAX(e.created_at),
		COUNT(DISTINCT CASE WHEN e.user_id <> '' THEN e.user_id END) FROM event e WHERE `+p.sql+" GROUP BY e.gid", p.args,
		func(r *sql.Rows, m *warnly.IssueMetrics) error {
			return r.Scan(&m.GID, &m.TimesSeen, &m.FirstSeen, &m.LastSeen, &m.UserCount)
		})
}

func (s *Store) CalculateEvents(ctx context.Context, c *warnly.ListIssueMetricsCriteria) ([]warnly.EventsPerHour, error) {
	p := active()
	p.period(c.From, c.To, false)
	in(&p, "e.pid", c.ProjectIDs)
	return collect(ctx, s.db, `SELECT e.event_hour, e.pid, COUNT(*) FROM event e WHERE `+p.sql+
		" GROUP BY e.event_hour, e.pid ORDER BY e.event_hour, e.pid LIMIT 5000", p.args,
		func(r *sql.Rows, m *warnly.EventsPerHour) error { return r.Scan(&m.TS, &m.ProjectID, &m.Count) })
}

func (s *Store) CalculateEventsPerDay(ctx context.Context, c *warnly.EventDefCriteria) ([]warnly.EventPerDay, error) {
	p := defPredicate(c)
	return collect(ctx, s.db, `SELECT e.gid, e.event_day, COUNT(*) FROM event e WHERE `+p.sql+
		" GROUP BY e.gid, e.event_day ORDER BY e.event_day DESC, e.gid", p.args,
		func(r *sql.Rows, m *warnly.EventPerDay) error { return r.Scan(&m.GID, &m.Time, &m.Count) })
}

func (s *Store) CountFields(ctx context.Context, c *warnly.EventDefCriteria) ([]warnly.FieldValueNum, error) {
	p := defPredicate(c)
	query := `SELECT tag_key, tag_value, n, first_seen, last_seen FROM (
		SELECT t.tag_key, t.tag_value, COUNT(*) n, MIN(e.created_at) first_seen, MAX(e.created_at) last_seen,
		ROW_NUMBER() OVER (PARTITION BY t.tag_key ORDER BY COUNT(*) DESC, t.tag_value) ranking` + tagJoin + p.sql +
		` GROUP BY t.tag_key, t.tag_value) ranked WHERE ranking <= 4 ORDER BY n DESC, tag_key, tag_value LIMIT 1000`
	return collect(ctx, s.db, query, p.args, func(r *sql.Rows, m *warnly.FieldValueNum) error {
		return r.Scan(&m.Tag, &m.Value, &m.Count, &m.FirstSeen, &m.LastSeen)
	})
}

func (s *Store) CalculateFields(ctx context.Context, c warnly.FieldsCriteria) ([]warnly.TagCount, error) {
	p := defPredicate(&warnly.EventDefCriteria{From: c.From, To: c.To, ProjectID: c.ProjectID, GroupID: c.IssueID})
	return s.tagCounts(ctx, p, 1000)
}

func (s *Store) ListPopularTags(ctx context.Context, c *warnly.ListPopularTagsCriteria) ([]warnly.TagCount, error) {
	p := active()
	p.period(c.From, c.To, true)
	in(&p, "e.pid", c.ProjectIDs)
	return s.tagCounts(ctx, p, c.Limit)
}

func (s *Store) ListTagValues(ctx context.Context, c *warnly.ListTagValuesCriteria) ([]warnly.TagValueCount, error) {
	p := active()
	p.period(c.From, c.To, true)
	in(&p, "e.pid", c.ProjectIDs)
	p.add("CAST(t.tag_key AS CHAR) = ?", c.Tag)
	return collect(ctx, s.db, "SELECT t.tag_value, COUNT(*) n"+tagJoin+p.sql+
		" GROUP BY t.tag_value ORDER BY n DESC, t.tag_value LIMIT ?", append(p.args, c.Limit),
		func(r *sql.Rows, m *warnly.TagValueCount) error { return r.Scan(&m.Value, &m.Count) })
}

func (s *Store) ListFieldFilters(ctx context.Context, c *warnly.FieldFilterCriteria) ([]warnly.Filter, error) {
	p := active()
	p.period(c.From, c.To, false)
	in(&p, "e.pid", c.ProjectIDs)
	return collect(ctx, s.db, "SELECT t.tag_key, t.tag_value"+tagJoin+p.sql+
		" GROUP BY t.tag_key, t.tag_value ORDER BY COUNT(*) DESC, t.tag_key, t.tag_value LIMIT 1000", p.args,
		func(r *sql.Rows, m *warnly.Filter) error { return r.Scan(&m.Key, &m.Value) })
}

func (s *Store) GetFilteredGroupIDs(
	ctx context.Context, tokens []warnly.QueryToken, from, to time.Time, projectIDs []int,
) ([]int64, error) {
	p := active()
	p.period(from, to, true)
	in(&p, "e.pid", projectIDs)
	for i := range tokens {
		if tokens[i].IsRawText {
			p.add("(LOCATE(LOWER(?), LOWER(e.message)) > 0 OR LOCATE(LOWER(?), LOWER(e.title)) > 0)", tokens[i].Value, tokens[i].Value)
		} else {
			p.tag(tokens[i].Key, tokens[i].Value, tokens[i].Operator == "is not")
		}
	}
	return collect(ctx, s.db, "SELECT DISTINCT e.gid FROM event e WHERE "+p.sql+" ORDER BY e.gid", p.args,
		func(r *sql.Rows, id *int64) error { return r.Scan(id) })
}

func (s *Store) GetEventPagination(ctx context.Context, c *warnly.EventPaginationCriteria) (*warnly.EventPagination, error) {
	id, err := canonicalID(c.EventID)
	if err != nil {
		return nil, err
	}

	p := active()
	p.period(c.From, c.To, true)
	p.add("e.pid = ? AND e.gid = ?", c.ProjectID, c.GroupID)

	result := &warnly.EventPagination{}
	for _, step := range []struct {
		target            *string
		order, comparison string
	}{
		{&result.FirstEventID, "DESC", ""},
		{&result.LastEventID, "ASC", ""},
		{&result.NextEventID, "DESC", "<"},
		{&result.PrevEventID, "ASC", ">"},
	} {
		q := predicate{sql: p.sql, args: append([]any(nil), p.args...)}
		if step.comparison != "" {
			q.add("(e.created_at "+step.comparison+" ? OR (e.created_at = ? AND e.event_id "+step.comparison+" ?))",
				c.CreatedAt.UTC(), c.CreatedAt.UTC(), id)
		}
		err := s.db.QueryRowContext(ctx, "SELECT e.event_id FROM event e WHERE "+q.sql+
			" ORDER BY e.created_at "+step.order+", e.event_id "+step.order+" LIMIT 1", q.args...).Scan(step.target)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return result, nil
}

func (s *Store) tagCounts(ctx context.Context, p predicate, limit int) ([]warnly.TagCount, error) {
	return collect(ctx, s.db, "SELECT t.tag_key, COUNT(*) n"+tagJoin+p.sql+
		" GROUP BY t.tag_key ORDER BY n DESC, t.tag_key LIMIT ?", append(p.args, limit),
		func(r *sql.Rows, m *warnly.TagCount) error { return r.Scan(&m.Tag, &m.Count) })
}
