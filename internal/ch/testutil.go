package ch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/vk-rv/warnly/internal/migrator"

	// Import the ClickHouse driver package for DSN parsing and connection.
	_ "github.com/ClickHouse/clickhouse-go/v2"
)

const (
	// databaseName is the name of the template database.
	databaseName = "test_db_template"

	// databaseUser and databasePassword are the credentials for testing.
	databaseUser     = "test_user"
	databasePassword = "testing123"

	// defaultClickHouseImageRef is the default database container to use.
	defaultClickHouseImageRef = "clickhouse/clickhouse-server:25.8"

	// clickhouseNativePort is the default TCP port for native protocol.
	clickhouseNativePort = "9000/tcp"
)

var testTimeout = 1 * time.Minute

// ApproxTime is a compare helper for clock skew.
var ApproxTime = cmp.Options{cmpopts.EquateApproxTime(1 * time.Second)}

// ClickHouseTestInstance is a wrapper around the Docker-based database instance.
type ClickHouseTestInstance struct {
	pool       *dockertest.Pool
	container  *dockertest.Resource
	url        *url.URL
	db         driver.Conn
	logger     *slog.Logger
	skipReason string
	dbLock     sync.Mutex
}

// MustTestInstance is NewTestInstance, except it prints errors to stderr and
// calls os.Exit when finished.
func MustTestInstance() *ClickHouseTestInstance {
	testDatabaseInstance, err := NewTestInstance()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return testDatabaseInstance
}

// NewTestInstance creates a new Docker-based ClickHouse instance. It creates an
// initial database, runs the migrations, and sets that database as a
// template.
func NewTestInstance() (*ClickHouseTestInstance, error) {
	if os.Getenv("INTEGRATION") == "" {
		return &ClickHouseTestInstance{
			skipReason: "🚧 Skipping database tests (INTEGRATION is not set)!",
		}, nil
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create the pool.
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to create database docker pool: %w", err)
	}

	// Determine the container image to use.
	repository, tag, err := clickHouseRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to determine database repository: %w", err)
	}

	// Start the actual container.
	runOptions := &dockertest.RunOptions{
		Repository: repository,
		Tag:        tag,
		Env: []string{
			"CLICKHOUSE_USER=" + databaseUser,
			"CLICKHOUSE_PASSWORD=" + databasePassword,
			"CLICKHOUSE_DB=" + databaseName,
		},
		ExposedPorts: []string{clickhouseNativePort},
	}
	container, err := pool.RunWithOptions(runOptions, func(c *docker.HostConfig) {
		c.AutoRemove = true
		c.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start database container: %w", err)
	}

	if err := container.Expire(120); err != nil {
		return nil, fmt.Errorf("failed to expire database container: %w", err)
	}

	// Get the host port for the native TCP port (9000).
	hostPort := container.GetHostPort(clickhouseNativePort)

	connectionURL := &url.URL{
		Scheme: "clickhouse",
		User:   url.UserPassword(databaseUser, databasePassword),
		Host:   hostPort,
		Path:   databaseName,
	}

	dsn := urlToClickHouseDSN(connectionURL)

	// Wait for the database to be ready and connect.
	db, _, err := ConnectLoop(ctx, dsn, testTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for database to be ready: %w", err)
	}

	// Run migrations on the template database.
	if err := dbMigrate(dsn, logger); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &ClickHouseTestInstance{
		pool:      pool,
		container: container,
		db:        db,
		url:       connectionURL,
		logger:    logger,
	}, nil
}

// MustClose is like Close except it prints the error to stderr and calls os.Exit.
func (i *ClickHouseTestInstance) MustClose() {
	if err := i.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Close terminates the test database instance, cleaning up any resources.
func (i *ClickHouseTestInstance) Close() (retErr error) {
	if i.skipReason != "" {
		return retErr
	}

	// Close database connection first
	if err := i.db.Close(); err != nil {
		retErr = fmt.Errorf("failed to close connection: %w", err)
		return retErr
	}

	// Purge the container
	if err := i.pool.Purge(i.container); err != nil {
		retErr = fmt.Errorf("failed to purge database container: %w", err)
		return retErr
	}

	return retErr
}

// NewDatabase creates a new database suitable for use in testing. It returns an
// established database connection (driver.Conn) and the configuration.
func (i *ClickHouseTestInstance) NewDatabase(tb testing.TB) driver.Conn {
	tb.Helper()

	if i.skipReason != "" {
		tb.Skip(i.skipReason)
	}

	// Create a new database and run migrations on it.
	newDatabaseName, err := i.createAndMigrate()
	if err != nil {
		tb.Fatal(err)
	}

	// Build the new connection URL for the new database name.
	connectionURL := *i.url
	connectionURL.Path = newDatabaseName
	connectionURL.RawQuery = ""

	dsn := urlToClickHouseDSN(&connectionURL)

	// Establish a connection to the database.
	ctx := tb.Context()
	db, closeDB, err := ConnectLoop(ctx, dsn, testTimeout, i.logger)
	if err != nil {
		tb.Fatalf("failed to connect to database %q: %s", newDatabaseName, err)
	}

	// Close connection and delete database when done.
	tb.Cleanup(func() {
		// Close connection first.
		if err := closeDB(); err != nil {
			tb.Errorf("failed to close database %q: %s", newDatabaseName, err)
		}

		// Drop the database. Execute the DROP statement directly on the main DB connection.
		q := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", newDatabaseName)

		i.dbLock.Lock()
		defer i.dbLock.Unlock()

		// Execute the DROP command using the main DB connection.
		// ClickHouse driver.Conn has an Exec method.
		if err := i.db.Exec(context.Background(), q); err != nil {
			tb.Errorf("failed to drop database %q: %s", newDatabaseName, err)
		}
	})

	return db
}

// createAndMigrate creates a new database with a random name and runs migrations.
func (i *ClickHouseTestInstance) createAndMigrate() (string, error) {
	name, err := randomDatabaseName()
	if err != nil {
		return "", fmt.Errorf("failed to generate random database name: %w", err)
	}

	ctx := context.Background()
	q := fmt.Sprintf("CREATE DATABASE `%s`;", name)

	i.dbLock.Lock()
	defer i.dbLock.Unlock()

	// Create the new database using the main connection pool.
	if err := i.db.Exec(ctx, q, nil); err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}

	// Build connection URL for the new database.
	connectionURL := *i.url
	connectionURL.Path = name
	connectionURL.RawQuery = ""

	dsn := urlToClickHouseDSN(&connectionURL)

	// Run migrations on the fresh database.
	if err := dbMigrate(dsn, i.logger); err != nil {
		// Attempt to drop the partially migrated database.
		dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", name)
		if err := i.db.Exec(ctx, dropQuery, nil); err != nil {
			return "", fmt.Errorf("failed to drop database: %w", err)
		}
		return "", fmt.Errorf("failed to migrate database: %w", err)
	}

	return name, nil
}

// dbMigrate runs the analytical migrations using the specific migrator.
func dbMigrate(dsn string, logger *slog.Logger) error {
	forceMigrate := true

	dbm, err := migrator.NewAnalyticsMigrator(dsn, logger)
	if err != nil {
		return err
	}

	if err = dbm.Up(forceMigrate); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}

	if sourceErr, err := dbm.Close(); sourceErr != nil || err != nil {
		return fmt.Errorf("close olap migrator: %w, %w", sourceErr, err)
	}

	return nil
}

// clickHouseRepo returns the container image name based on an
// environment variable, or the default value if the environment variable is
// unset.
func clickHouseRepo() (string, string, error) {
	ref := os.Getenv("CI_CLICKHOUSE_IMAGE")
	if ref == "" {
		ref = defaultClickHouseImageRef
	}

	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid reference for database container: %q", ref)
	}
	return parts[0], parts[1], nil
}

// randomDatabaseName returns a random database name.
func randomDatabaseName() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "test_" + hex.EncodeToString(b), nil
}

// urlToClickHouseDSN converts a URL to a ClickHouse DSN (e.g., for clickhouse-go/v2).
// Format: clickhouse://[user]:[password]@[host]/[database]
func urlToClickHouseDSN(u *url.URL) string {
	password, _ := u.User.Password()
	dbName := strings.TrimPrefix(u.Path, "/")

	// DSN format for clickhouse-go/v2: clickhouse://username:password@host/dbname?params
	return fmt.Sprintf("clickhouse://%s:%s@%s/%s%s",
		u.User.Username(),
		password,
		u.Host,
		dbName,
		func() string {
			if u.RawQuery != "" {
				return "?" + u.RawQuery
			}
			return ""
		}(),
	)
}
