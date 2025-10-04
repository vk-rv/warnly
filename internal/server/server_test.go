package server_test

import (
	"testing"

	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/mysql"
)

var (
	testMySQLDatabaseInstance      *mysql.TestInstance
	testClickHouseDatabaseInstance *ch.ClickHouseTestInstance
)

func TestMain(m *testing.M) {
	testMySQLDatabaseInstance = mysql.MustTestInstance()
	defer testMySQLDatabaseInstance.MustClose()

	testClickHouseDatabaseInstance = ch.MustTestInstance()
	defer testClickHouseDatabaseInstance.MustClose()

	m.Run()
}
