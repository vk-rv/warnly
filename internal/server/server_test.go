package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/patrickmn/go-cache"
	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/server"
	"github.com/vk-rv/warnly/internal/svcotel"
	"github.com/vk-rv/warnly/internal/uow"
	"github.com/vk-rv/warnly/internal/warnly"
)

const projectDetailsPath = "/projects/{id}?issues=all&period=1h"

const (
	testBaseURL    = "http://localhost:8030"
	testBaseScheme = "http"
)

const (
	testTeamName     = "go-test"
	testOwnerID      = 1
	testProjectName  = "go-test"
	testProjectKey   = "urzovxt"
	testProjectIDStr = "1"
	testProjectIDKey = "project_id"
)

var testSentryAuthHeader = "Sentry sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=" + testProjectKey

var (
	testMySQLDatabaseInstance      *mysql.TestInstance
	testClickHouseDatabaseInstance *ch.ClickHouseTestInstance
)

var testUser = warnly.User{
	ID:       1,
	Name:     "John",
	Surname:  "Doe",
	Username: "johndoe",
	Email:    "johndoe@example.com",
}

//nolint:unused // will be used in tests in cookie store
var testKey = []byte("01234567890123456789012345678901")

func TestMain(m *testing.M) {
	testMySQLDatabaseInstance = mysql.MustTestInstance()
	defer testMySQLDatabaseInstance.MustClose()

	testClickHouseDatabaseInstance = ch.MustTestInstance()
	defer testClickHouseDatabaseInstance.MustClose()

	m.Run()
}

type testStores struct {
	projectStore    warnly.ProjectStore
	assingmentStore warnly.AssingmentStore
	messageStore    warnly.MessageStore
	mentionStore    warnly.MentionStore
	teamStore       warnly.TeamStore
	issueStore      warnly.IssueStore
	memoryCache     *cache.Cache
	olap            *ch.ClickhouseStore
	uow             uow.StartUnitOfWork
}

func getTestStores(testDB *sql.DB, testOlapDB clickhouse.Conn, logger *slog.Logger) testStores {
	olap := ch.NewClickhouseStore(testOlapDB, svcotel.NewNoopProvider())
	olap.EnableAsyncInsertWait()
	return testStores{
		projectStore:    mysql.NewProjectStore(testDB),
		assingmentStore: mysql.NewAssingmentStore(testDB),
		messageStore:    mysql.NewMessageStore(testDB),
		mentionStore:    mysql.NewMentionStore(testDB),
		teamStore:       mysql.NewTeamStore(testDB),
		issueStore:      mysql.NewIssueStore(testDB),
		memoryCache:     cache.New(5*time.Minute, 10*time.Minute),
		olap:            olap,
		uow:             mysql.NewUOW(testDB, logger),
	}
}

func getTestLogger() (*slog.Logger, bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	return logger, buf
}

func getIngestRequest(body []byte) (*httptest.ResponseRecorder, *http.Request) {
	r := httptest.NewRequest(
		http.MethodPost,
		ingestEventPath,
		bytes.NewReader(body))
	r.SetPathValue(testProjectIDKey, testProjectIDStr)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Sentry-Auth", testSentryAuthHeader)

	w := httptest.NewRecorder()

	return w, r
}

func getProjectDetailsRequest(ctx context.Context) (*httptest.ResponseRecorder, *http.Request) {
	r := httptest.NewRequest(http.MethodGet, projectDetailsPath, http.NoBody)
	r.SetPathValue("id", "1")
	r = r.WithContext(server.NewContextWithUser(ctx, testUser))

	w := httptest.NewRecorder()

	return w, r
}
