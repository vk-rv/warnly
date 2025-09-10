// Package migrator provides functionality to manage database migrations.
// It supports MySQL and ClickHouse databases, performing schema migrations
// using the golang-migrate library.
package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	clickhouseMigrate "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/vk-rv/warnly/migrations"
)

const (
	_ Driver = iota
	MySQL
	Clickhouse
)

var expectedVersions = map[Driver]uint{
	MySQL:      0,
	Clickhouse: 0,
}

var driverToString = map[Driver]string{
	MySQL:      "mysql",
	Clickhouse: "clickhouse",
}

// Driver represents a database driver.
type Driver uint8

// String returns the string representation of the driver.
func (d Driver) String() string {
	return driverToString[d]
}

// Migrator is responsible for migrating the database schema.
type Migrator struct {
	db       *sql.DB
	migrator *migrate.Migrate
	logger   *slog.Logger
	driver   Driver
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(dsn string, logger *slog.Logger) (*Migrator, error) {
	const mysqlDriver = "mysql"

	if !strings.Contains(dsn, "multiStatements=true") {
		dsn += "&multiStatements=true"
	}

	db, err := sql.Open(mysqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate: open a database specified by its database driver: %w", err)
	}

	dr, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("migrate: mysql with instance: %w", err)
	}

	sourceDriver, err := iofs.New(migrations.FS, mysqlDriver)
	if err != nil {
		return nil, err
	}

	mm, err := migrate.NewWithInstance("iofs", sourceDriver, mysqlDriver, dr)
	if err != nil {
		return nil, fmt.Errorf("migrate: creating migrate instance: %w", err)
	}

	return &Migrator{
		db:       db,
		migrator: mm,
		logger:   logger,
		driver:   MySQL,
	}, nil
}

// NewAnalyticsMigrator returns a migrator for analytics databases.
func NewAnalyticsMigrator(dsn string, logger *slog.Logger) (*Migrator, error) {
	const clickhouseDriver = "clickhouse"

	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("migrator: parse clickhouse dsn: %w", err)
	}
	opts.MaxIdleConns = 0
	opts.MaxOpenConns = 0
	opts.ConnMaxLifetime = 0

	db := clickhouse.OpenDB(opts)

	dr, err := clickhouseMigrate.WithInstance(db, &clickhouseMigrate.Config{
		MigrationsTableEngine: "MergeTree",
		MultiStatementEnabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf("migrator: getting db driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrations.FS, clickhouseDriver)
	if err != nil {
		return nil, fmt.Errorf("migrator: creating iofs source driver: %w", err)
	}

	mm, err := migrate.NewWithInstance("iofs", sourceDriver, clickhouseDriver, dr)
	if err != nil {
		return nil, fmt.Errorf("creating migrate instance: %w", err)
	}

	return &Migrator{
		db:       db,
		migrator: mm,
		logger:   logger,
		driver:   Clickhouse,
	}, nil
}

// Close closes the source and db.
func (m *Migrator) Close() (source, db error) {
	return m.migrator.Close()
}

// Up runs any pending migrations.
// if canAutoMigrate is false and there are pending migrations, an error is returned
// for manual safety.
func (m *Migrator) Up(canAutoMigrate bool) error {
	// check if any migrations are pending
	currentVersion, _, err := m.migrator.Version()
	if err != nil {
		if !errors.Is(err, migrate.ErrNilVersion) {
			return fmt.Errorf("migrator: getting current migrations version: %w", err)
		}

		m.logger.Info("migrator: first run, running migrations...")

		// if first run then it's safe to migrate
		if err := m.migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("running migrations: %w", err)
		}

		m.logger.Info("migrator: migrations complete")

		return nil
	}

	expectedVersion := expectedVersions[m.driver]

	if currentVersion < expectedVersion {
		if !canAutoMigrate {
			return errors.New(`migrator: migrations pending, 
				please set FORCE_MIGRATE to true 
				or backup your database and run migrations manually`)
		}

		m.logger.Info("migrator: current migration",
			slog.Uint64("current_version", uint64(currentVersion)),
			slog.Uint64("expected_version", uint64(expectedVersion)))

		m.logger.Info("migrator: running migrations...")

		if err := m.migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("running migrations: %w", err)
		}

		m.logger.Info("migrator: migrations complete")

		return nil
	}

	m.logger.Info("migrator: migrations up to date")

	return nil
}

// Drop drops the database.
func (m *Migrator) Drop(ctx context.Context) error {
	m.logger.Debug("migrator: running drop ...")

	_, err := m.db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0;")
	if err != nil {
		return fmt.Errorf("migrator: setting foreign key checks to 0: %w", err)
	}

	if err := m.migrator.Drop(); err != nil {
		return fmt.Errorf("migrator dropping: %w", err)
	}

	m.logger.Debug("migrator: drop complete")

	return nil
}
