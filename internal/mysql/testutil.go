package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/vk-rv/warnly/internal/migrator"
)

const (
	// databaseName is the name of the template database to clone.
	databaseName = "test-db-template"

	// databaseUser and databasePassword are the username and password for
	// connecting to the database. These values are only used for testing.
	databaseUser     = "test-user"
	databasePassword = "testing123"

	rootPassword = "rootpassword123"

	// defaultMySQLImageRef is the default database container to use if none is
	// specified.
	defaultMySQLImageRef = "mysql:9.4.0"
)

// ApproxTime is a compare helper for clock skew.
var ApproxTime = cmp.Options{cmpopts.EquateApproxTime(1 * time.Second)}

// TestInstance is a wrapper around the Docker-based database instance.
type TestInstance struct {
	pool       *dockertest.Pool
	container  *dockertest.Resource
	db         *sql.DB
	rootDB     *sql.DB
	url        *url.URL
	logger     *slog.Logger
	skipReason string
	dbConfig   DBConfig
	dbLock     sync.Mutex
}

// MustTestInstance is NewTestInstance, except it prints errors to stderr and
// calls os.Exit when finished. Callers can call Close or MustClose().
func MustTestInstance() *TestInstance {
	testDatabaseInstance, err := NewTestInstance()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return testDatabaseInstance
}

// NewTestInstance creates a new Docker-based database instance. It also creates
// an initial database, runs the migrations, and sets that database as a
// template to be cloned by future tests.
//
// This should not be used outside of testing, but it is exposed in the package
// so it can be shared with other packages. It should be called and instantiated
// in TestMain.
func NewTestInstance() (*TestInstance, error) {
	if os.Getenv("INTEGRATION") == "" {
		return &TestInstance{
			skipReason: "🚧 Skipping database tests (INTEGRATION is not set)!",
		}, nil
	}

	ctx := context.Background()

	// Create the pool.
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to create database docker pool: %w", err)
	}

	// Determine the container image to use.
	repository, tag, err := mysqlRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to determine database repository: %w", err)
	}

	// Start the actual container.
	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repository,
		Tag:        tag,
		Env: []string{
			"MYSQL_ROOT_PASSWORD=" + rootPassword,
			"MYSQL_DATABASE=" + databaseName,
			"MYSQL_USER=" + databaseUser,
			"MYSQL_PASSWORD=" + databasePassword,
		},
		ExposedPorts: []string{"3306/tcp"},
	}, func(c *docker.HostConfig) {
		c.AutoRemove = true
		c.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start database container: %w", err)
	}

	// Stop the container after its been running for too long. No test suite
	// should take super long.
	if err := container.Expire(120); err != nil {
		return nil, fmt.Errorf("failed to expire database container: %w", err)
	}

	hostPort := container.GetHostPort("3306/tcp")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	rootDSN := fmt.Sprintf("root:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4",
		rootPassword,
		hostPort,
		databaseName,
	)
	rootConfig := DBConfig{
		DSN:     rootDSN,
		Timeout: 120 * time.Second,
		PoolConfig: PoolConfig{
			maxOpenConnections: 2,
			maxIdleConnections: 2,
			maxLifetime:        5 * time.Minute,
		},
	}
	rootDB, _, err := ConnectLoop(ctx, rootConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect as root: %w", err)
	}

	// Grant all privileges to the test user on all databases they create
	grantQuery := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%%`.* TO '%s'@'%%';", databaseUser)
	if _, err := rootDB.ExecContext(ctx, grantQuery); err != nil {
		return nil, fmt.Errorf("failed to grant privileges: %w", err)
	}

	connectionURL := &url.URL{
		Scheme:   "mysql",
		User:     url.UserPassword(databaseUser, databasePassword),
		Host:     hostPort,
		Path:     databaseName,
		RawQuery: "parseTime=true&charset=utf8mb4",
	}

	dsn := urlToMySQLDSN(connectionURL)

	dbConfig := DBConfig{
		DSN:     dsn,
		Timeout: 120 * time.Second,
		PoolConfig: PoolConfig{
			maxOpenConnections: 5,
			maxIdleConnections: 5,
			maxLifetime:        5 * time.Minute,
		},
	}

	db, _, err := ConnectLoop(ctx, dbConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for database to be ready: %w", err)
	}

	if err := dbMigrate(dsn, logger); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &TestInstance{
		pool:      pool,
		container: container,
		db:        db,
		rootDB:    rootDB,
		url:       connectionURL,
		dbConfig:  dbConfig,
		logger:    logger,
	}, nil
}

// MustClose is like Close except it prints the error to stderr and calls os.Exit.
func (i *TestInstance) MustClose() error {
	if err := i.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return nil
}

// Close terminates the test database instance, cleaning up any resources.
func (i *TestInstance) Close() (retErr error) {
	// Do not attempt to close  things when there's nothing to close.
	if i.skipReason != "" {
		return retErr
	}

	defer func() {
		if err := i.pool.Purge(i.container); err != nil {
			retErr = fmt.Errorf("failed to purge database container: %w", err)
			return
		}
	}()

	if err := i.db.Close(); err != nil {
		retErr = fmt.Errorf("failed to close connection: %w", err)
		return retErr
	}

	if err := i.rootDB.Close(); err != nil {
		retErr = fmt.Errorf("failed to close root connection: %w", err)
		return retErr
	}

	return retErr
}

// NewDatabase creates a new database suitable for use in testing. It returns an
// established database connection and the configuration.
func (i *TestInstance) NewDatabase(tb testing.TB) (*sql.DB, DBConfig) {
	tb.Helper()

	// Ensure we should actually create the database.
	if i.skipReason != "" {
		tb.Skip(i.skipReason)
	}

	// Clone the template database.
	newDatabaseName, err := i.clone()
	if err != nil {
		tb.Fatal(err)
	}

	// Build the new connection URL for the new database name. Query params are
	// dropped with ResolveReference, so we have to re-add disabling SSL over
	// localhost.
	connectionURL := i.url.ResolveReference(&url.URL{Path: newDatabaseName})
	connectionURL.RawQuery = "parseTime=true&charset=utf8mb4"

	dsn := urlToMySQLDSN(connectionURL)

	dbConfig := DBConfig{
		DSN:     dsn,
		Timeout: 1 * time.Second,
		PoolConfig: PoolConfig{
			maxOpenConnections: 5,
			maxIdleConnections: 5,
			maxLifetime:        5 * time.Minute,
		},
	}

	// Establish a connection to the database.
	ctx := tb.Context()
	db, closeDB, err := ConnectLoop(ctx, dbConfig, i.logger)
	if err != nil {
		tb.Fatalf("failed to connect to database %q: %s", newDatabaseName, err)
	}

	// Close connection and delete database when done.
	tb.Cleanup(func() {
		// Close connection first. It is an error to drop a database with active
		// connections.
		if err := closeDB(); err != nil {
			tb.Errorf("failed to close database %q: %s", newDatabaseName, err)
		}

		// Drop the database to keep the container from running out of resources.
		q := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", newDatabaseName)

		i.dbLock.Lock()
		defer i.dbLock.Unlock()

		if _, err := i.rootDB.ExecContext(context.Background(), q); err != nil {
			tb.Errorf("failed to drop database %q: %s", newDatabaseName, err)
		}
	})

	return db, dbConfig
}

// clone creates a new database with a random name from the template instance using MySQL syntax.
// It also runs migrations on the new database.
func (i *TestInstance) clone() (string, error) {
	// Generate a random database name.
	name, err := randomDatabaseName()
	if err != nil {
		return "", fmt.Errorf("failed to generate random database name: %w", err)
	}

	ctx := context.Background()
	q := fmt.Sprintf("CREATE DATABASE `%s`;", name)

	i.dbLock.Lock()
	defer i.dbLock.Unlock()

	// Create the new database.
	if _, err := i.rootDB.ExecContext(ctx, q); err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}

	// Build connection URL for the new database.
	connectionURL := *i.url
	connectionURL.Path = name
	connectionURL.RawQuery = "parseTime=true&charset=utf8mb4"

	dsn := urlToMySQLDSN(&connectionURL)

	if err := dbMigrate(dsn, i.logger); err != nil {
		return "", fmt.Errorf("failed to migrate database: %w", err)
	}

	return name, nil
}

// dbMigrate runs the migrations. dsn is the connection URL string.
func dbMigrate(dsn string, logger *slog.Logger) error {
	forceMigrate := true

	dbm, err := migrator.NewMigrator(dsn, logger)
	if err != nil {
		return err
	}

	if err = dbm.Up(forceMigrate); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}

	if sourceErr, err := dbm.Close(); sourceErr != nil || err != nil {
		return fmt.Errorf("close oltp migrator: %w, %w", sourceErr, err)
	}

	return nil
}

// mysqlRepo returns the mysql container image name based on an
// environment variable, or the default value if the environment variable is
// unset.
func mysqlRepo() (string, string, error) {
	ref := os.Getenv("CI_MYSQL_IMAGE")
	if ref == "" {
		ref = defaultMySQLImageRef
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
	return hex.EncodeToString(b), nil
}

// Helper function to convert URL to MySQL DSN (optional, for consistency).
func urlToMySQLDSN(u *url.URL) string {
	password, _ := u.User.Password()
	dbName := strings.TrimPrefix(u.Path, "/")

	// MySQL DSN format: username:password@tcp(host)/dbname?params
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		u.User.Username(),
		password,
		u.Host,
		dbName,
		u.RawQuery,
	)
}
