// Package ch provides a ClickHouse connection utility.
package ch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const DefaultTimeout = 5 * time.Second

// ConnectLoop tries to connect to ClickHouse with retries until the defaultTimeout is reached.
// It returns the connection and a close function to close the connection pool.
//
//nolint:ireturn // return external client.
func ConnectLoop(
	ctx context.Context,
	dsn string,
	connTimeout time.Duration,
	logger *slog.Logger,
) (conn driver.Conn, closeFunc func() error, err error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("parse clickhouse dsn: %w", err)
	}

	db, err := clickhouse.Open(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("open clickhouse connection pool: %w", err)
	}

	if err = db.Ping(ctx); err == nil {
		return db, db.Close, nil
	}

	logger.Error("clickhouse: ping db problem", slog.Any("error", err))

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutExceeded := time.After(connTimeout)

	for {
		select {
		case <-timeoutExceeded:
			return nil, nil, fmt.Errorf("clickhouse: db connection failed after %s timeout", connTimeout)
		case <-ticker.C:
			if err = db.Ping(ctx); err == nil {
				return db, db.Close, nil
			}
			logger.Error("clickhose: ping db problem", slog.Any("error", err))
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("clickhouse: db connection failed, ctx done: %w", ctx.Err())
		}
	}
}
