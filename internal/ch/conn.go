// Package ch provides a ClickHouse connection utility.
package ch

import (
	"context"
	"errors"
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

	if !isRetryableError(err) {
		return nil, nil, err
	}

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
			if !isRetryableError(err) {
				return nil, nil, err
			}
			logger.Error("clickhouse: ping db problem", slog.Any("error", err))
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("clickhouse: db connection failed, ctx done: %w", ctx.Err())
		}
	}
}

// isRetryableError determines if an error is transient and worth retrying.
// Returns true for network/timeout errors, false for auth/config errors.
func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check ClickHouse specific errors
	var chErr *clickhouse.Exception
	if errors.As(err, &chErr) {
		return isClickHouseErrorRetryable(chErr)
	}

	// Check network errors
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var netOpErr interface{ Temporary() bool }
	if errors.As(err, &netOpErr) && netOpErr.Temporary() {
		return true
	}

	return false
}

// isClickHouseErrorRetryable checks ClickHouse specific error codes.
func isClickHouseErrorRetryable(err *clickhouse.Exception) bool {
	switch err.Code {
	// Authentication and permission errors - NOT retryable
	case 192, // UNKNOWN_USER
		193, // WRONG_PASSWORD
		516, // AUTHENTICATION_FAILED
		497, // UNKNOWN_DATABASE
		241, // UNKNOWN_TABLE
		60,  // ACCESS_DENIED
		164: // READONLY
		return false

	// Connection/resource errors - retryable
	case 159, // TIMEOUT_EXCEEDED
		209, // SOCKET_TIMEOUT
		210, // NETWORK_ERROR
		425, // CANNOT_CONNECT_RABBITMQ (network issue)
		279, // ALL_CONNECTION_TRIES_FAILED
		242: // TOO_MANY_SIMULTANEOUS_QUERIES
		return true

	default:
		// Unknown ClickHouse errors - don't retry
		return false
	}
}
