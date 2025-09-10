// Package mysql provides a connection to MySQL database with a connection pool.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	cfg.Timeout = time.Second * 3

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
			logger.Error("mysql: connect to the database", slog.Any("error", err))
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("mysql: db connection failed, ctx done: %w", ctx.Err())
		}
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
