// Package mysql provides a connection to MySQL database with a connection pool.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
)

// SQLogger is a logger for mysql.
type SQLogger struct {
	Logger *slog.Logger
}

// Print is a method for logging adapter.
func (l *SQLogger) Print(v ...any) {
	line := fmt.Sprintln(v...)
	l.Logger.Info(line)
}

// DBConfig contains information sufficient for database connection.
type DBConfig struct {
	DSN        string        `env:"DB_DSN"      env-default:"username:password@protocol(address)/dbname?param=value" yaml:"dsn"`
	PoolConfig PoolConfig    `yaml:"poolConfig"`
	Timeout    time.Duration `yaml:"timeout"` // timeout for trying to connect to the database
}

// PoolConfig is a db pool configuration.
type PoolConfig struct {
	maxOpenConnections int
	maxIdleConnections int
	maxLifetime        time.Duration
}

// ConnectLoop takes config and specified database credentials as input, returning *sql.DB handle for interactions
// with database.
// It tries to connect to the database until timeout is exceeded to handle cases
// when database is not ready yet (e.g. in docker-compose setups).
func ConnectLoop(ctx context.Context, cfg DBConfig, logger *slog.Logger) (db *sql.DB, closeFunc func() error, err error) {
	cfg.PoolConfig.maxOpenConnections = 20
	cfg.PoolConfig.maxIdleConnections = 20
	cfg.PoolConfig.maxLifetime = 5 * time.Minute

	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second * 3
	}

	dsn := cfg.DSN
	const driverName = "mysql"

	if err = mysql.SetLogger(&SQLogger{logger.With("subsystem", driverName)}); err != nil {
		return nil, nil, errors.New("mysql: problem setting logger")
	}

	db, err = createDBPool(ctx, driverName, dsn)
	if err == nil {
		configureDBPool(db, cfg.PoolConfig)
		return db, db.Close, nil
	}

	logger.Error("mysql: failed to connect to the database", slog.Any("error", err))

	if !isRetryableError(err) {
		return nil, nil, err
	}

	if cfg.Timeout == 0 {
		const defaultTimeout = 5
		cfg.Timeout = defaultTimeout
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutExceeded := time.After(cfg.Timeout)

	for {
		select {
		case <-timeoutExceeded:
			return nil, nil, fmt.Errorf("mysql: db connection failed after %s timeout", cfg.Timeout)
		case <-ticker.C:
			db, err := createDBPool(ctx, driverName, dsn)
			if err == nil {
				configureDBPool(db, cfg.PoolConfig)
				return db, db.Close, nil
			}
			if !isRetryableError(err) {
				return nil, nil, err
			}
			logger.Error("mysql: connect to the database", slog.Any("error", err))
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("mysql: db connection failed, ctx done: %w", ctx.Err())
		}
	}
}

// isRetryableError determines if an error is transient and worth retrying.
// Returns true for network/timeout errors, false for auth/config errors.
func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, mysql.ErrInvalidConn) || errors.Is(err, io.EOF) {
		return true
	}

	var errno syscall.Errno
	if errors.As(err, &errno) {
		return isSyscallErrorRetryable(errno)
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return isMySQLErrorRetryable(mysqlErr)
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

// isMySQLErrorRetryable checks MySQL specific error codes.
func isMySQLErrorRetryable(err *mysql.MySQLError) bool {
	switch err.Number {
	// Authentication and permission errors - NOT retryable
	case 1044, // ER_DBACCESS_DENIED_ERROR
		1045, // ER_ACCESS_DENIED_ERROR
		1049, // ER_BAD_DB_ERROR
		1095, // ER_KILL_DENIED_ERROR
		1142, // ER_TABLEACCESS_DENIED_ERROR
		1143, // ER_COLUMNACCESS_DENIED_ERROR
		1227, // ER_SPECIFIC_ACCESS_DENIED_ERROR
		1370, // ER_PROC_AUTO_GRANT_FAIL
		1396: // ER_CANNOT_USER
		return false

	// Connection/resource errors - retryable
	case 1040, // ER_CON_COUNT_ERROR - Too many connections
		1152, // ER_ABORTING_CONNECTION
		1153, // ER_NET_PACKET_TOO_LARGE
		1159, // ER_NET_READ_ERROR
		1160, // ER_NET_READ_INTERRUPTED
		1161: // ER_NET_ERROR_ON_WRITE
		return true

	default:
		// Unknown MySQL errors - don't retry
		return false
	}
}

// isSyscallErrorRetryable checks if a syscall error is retryable.
//
//nolint:exhaustive // consider adding more error codes if needed
func isSyscallErrorRetryable(errno syscall.Errno) bool {
	switch errno {
	case syscall.ECONNREFUSED, // Connection refused
		syscall.ECONNRESET,   // Connection reset
		syscall.ECONNABORTED, // Connection aborted
		syscall.ETIMEDOUT,    // Timeout
		syscall.EHOSTUNREACH, // Host unreachable
		syscall.ENETUNREACH:  // Network unreachable
		return true
	default:
		return false
	}
}

// createDBPool creates pool of connections to sql server and pings db under the hood.
func createDBPool(ctx context.Context, driverName, dsn string) (*sql.DB, error) {
	db, err := otelsql.Open(
		driverName,
		dsn,
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		otelsql.WithDBName("oltp"))
	if err != nil {
		return nil, fmt.Errorf("db: otelsql open primary db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("mysql: ping database: %w", err)
	}

	return db, nil
}

// configureDBPool just sets up a database connection pool.
func configureDBPool(db *sql.DB, config PoolConfig) {
	db.SetMaxOpenConns(config.maxOpenConnections)
	db.SetMaxIdleConns(config.maxIdleConnections)
	db.SetConnMaxLifetime(config.maxLifetime)
}
