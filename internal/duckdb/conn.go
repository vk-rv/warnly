// Package duckdb implements analytics over MySQL's ENGINE=DuckDB.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/vk-rv/warnly/internal/mysql"
)

// NormalizeDSN enables text-protocol queries, which the engine can push down,
// and ensures all DATETIME values are written and read in UTC.
func NormalizeDSN(dsn string) (string, error) {
	if dsn == "" {
		return "", errors.New("mysql-duckdb: ANALYTICS_DSN is required")
	}
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("mysql-duckdb: parse DSN: %w", err)
	}
	if cfg.DBName == "" {
		return "", errors.New("mysql-duckdb: DSN must specify an analytics database")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.InterpolateParams = true
	if err := cfg.Apply(driver.Charset("utf8mb4", "utf8mb4_bin")); err != nil {
		return "", fmt.Errorf("mysql-duckdb: configure charset: %w", err)
	}
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	cfg.Params["time_zone"] = "'+00:00'"
	cfg.Params["collation_connection"] = "'utf8mb4_0900_bin'"
	
	return cfg.FormatDSN(), nil
}

// ConnectLoop connects to MySQL and refuses servers without the DuckDB engine.
func ConnectLoop(ctx context.Context, dsn string, logger *slog.Logger) (*sql.DB, func() error, error) {
	dsn, err := NormalizeDSN(dsn)
	if err != nil {
		return nil, nil, err
	}
	
	db, closeDB, err := mysql.ConnectLoop(ctx, mysql.DBConfig{DSN: dsn, Timeout: 30 * time.Second}, logger)
	if err != nil {
		return nil, nil, err
	}
	
	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.ENGINES
		WHERE ENGINE = 'DuckDB' AND SUPPORT IN ('YES', 'DEFAULT')`).Scan(&count)
	if err != nil || count != 1 {
		_ = closeDB()
		if err != nil {
			return nil, nil, fmt.Errorf("mysql-duckdb: check engine: %w", err)
		}
		return nil, nil, errors.New("mysql-duckdb: server does not support ENGINE=DuckDB")
	}
	
	return db, closeDB, nil
}
