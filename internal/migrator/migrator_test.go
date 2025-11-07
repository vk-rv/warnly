package migrator_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/migrator"
	"github.com/vk-rv/warnly/internal/mysql"
)

var (
	testMySQLInstance      *mysql.TestInstance
	testClickHouseInstance *ch.ClickHouseTestInstance
)

func TestMain(m *testing.M) {
	testMySQLInstance = mysql.MustTestInstance()
	defer testMySQLInstance.MustClose()

	testClickHouseInstance = ch.MustTestInstance()
	defer testClickHouseInstance.MustClose()

	m.Run()
}

func getTestLogger() (*slog.Logger, bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	return logger, buf
}

func TestDriver(t *testing.T) {
	t.Parallel()

	t.Run("MySQL driver string", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "mysql", migrator.MySQL.String())
	})

	t.Run("ClickHouse driver string", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "clickhouse", migrator.Clickhouse.String())
	})
}

func TestMigrator_Up(t *testing.T) {
	t.Parallel()

	t.Run("successful migration on fresh MySQL database", func(t *testing.T) {
		t.Parallel()

		logger, _ := getTestLogger()

		testDB, dbConfig := testMySQLInstance.NewDatabase(t)
		defer testDB.Close()

		m, err := migrator.NewMigrator(dbConfig.DSN, logger)
		require.NoError(t, err)
		defer func() {
			sourceErr, dbErr := m.Close()
			assert.NoError(t, sourceErr)
			assert.NoError(t, dbErr)
		}()

		// First run should succeed
		err = m.Up(true)
		require.NoError(t, err)

		// Second run should be no-op
		err = m.Up(true)
		assert.NoError(t, err)
	})

	t.Run("migration with pending migrations and auto-migrate disabled", func(t *testing.T) {
		t.Parallel()

		logger, _ := getTestLogger()

		testDB, dbConfig := testMySQLInstance.NewDatabase(t)
		defer testDB.Close()

		m, err := migrator.NewMigrator(dbConfig.DSN, logger)
		require.NoError(t, err)
		defer func() {
			sourceErr, dbErr := m.Close()
			assert.NoError(t, sourceErr)
			assert.NoError(t, dbErr)
		}()

		err = m.Up(false)
		assert.NoError(t, err)
	})
}

func TestMigrator_Drop(t *testing.T) {
	t.Parallel()

	t.Run("successful drop MySQL database", func(t *testing.T) {
		t.Parallel()

		logger, _ := getTestLogger()

		testDB, dbConfig := testMySQLInstance.NewDatabase(t)
		defer testDB.Close()

		m, err := migrator.NewMigrator(dbConfig.DSN, logger)
		require.NoError(t, err)
		defer func() {
			sourceErr, dbErr := m.Close()
			assert.NoError(t, sourceErr)
			assert.NoError(t, dbErr)
		}()

		err = m.Drop(t.Context())
		assert.NoError(t, err)
	})
}
