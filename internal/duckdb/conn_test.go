package duckdb_test

import (
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/duckdb"
)

func TestNormalizeDSN(t *testing.T) {
	t.Parallel()
	for _, dsn := range []string{"", "not a dsn", "user:pass@tcp(localhost:3306)/"} {
		_, err := duckdb.NormalizeDSN(dsn)
		require.Error(t, err)
	}
	dsn, err := duckdb.NormalizeDSN("user:pass@tcp(localhost:3306)/analytics?loc=Local&parseTime=false&interpolateParams=false")
	require.NoError(t, err)
	cfg, err := driver.ParseDSN(dsn)
	require.NoError(t, err)
	require.True(t, cfg.ParseTime)
	require.True(t, cfg.InterpolateParams)
	require.Equal(t, time.UTC, cfg.Loc)
	require.Equal(t, "utf8mb4_bin", cfg.Collation)
	require.Equal(t, "'utf8mb4_0900_bin'", cfg.Params["collation_connection"])
	require.Equal(t, "'+00:00'", cfg.Params["time_zone"])
	require.Equal(t, "analytics", cfg.DBName)
}
