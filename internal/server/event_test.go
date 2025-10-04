package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/ch"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/server"
	"github.com/vk-rv/warnly/internal/svc/event"
	"github.com/vk-rv/warnly/internal/svcotel"
	"github.com/vk-rv/warnly/internal/warnly"
)

const ingestEventPath = "/ingest/api/{project_id}/envelope/"

var testTime = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

var body = []byte(`{"event_id":"3708a788c39c44508a3c9442214b2f9f","sent_at":"2025-10-04T02:33:58.305163+03:00","dsn":"http://urzovxt@127.0.0.1:8030/ingest/2","sdk":{"name":"sentry.go","version":"0.30.0"},"trace":{"environment":"production","public_key":"urzovxt","release":"1.0.0","trace_id":"588ec04f1a82873ba26825cc1d8594d9"}}
{"type":"event","length":3663}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":5,"name":"go","version":"go1.24.1"},"trace":{"span_id":"01c4097208d35070","trace_id":"588ec04f1a82873ba26825cc1d8594d9"}},"environment":"production","event_id":"3708a788c39c44508a3c9442214b2f9f","level":"error","message":"My error message at 2025-10-04T02:33:58+03:00","platform":"go","release":"1.0.0","sdk":{"name":"sentry.go","version":"0.30.0","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.30.0"}]},"server_name":"MacBook-Pro.local","threads":[{"stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoe/game/cmd/gameproj/main.go","lineno":95,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"gameproj web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoe/game/cmd/gameproj/main.go","lineno":149,"pre_context":["","\tfor {","\t\ttime.Sleep(time.Second * 10)","\t\t// randomize error message","\t\tnow := time.Now()"],"context_line":"\t\tmyLogger.Error(fmt.Sprintf(\"My error message at %s\", now.Format(time.RFC3339)))","post_context":["\t\tsentryLogger.Error(fmt.Sprintf(\"My info message at %s\", now.Format(time.RFC3339)))","","\t\terr := fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))","\t\tmyLogger.Error(\"An example error occurred\", zap.Error(err))","\t\tsentryLogger.Error(\"An example info message\", zap.Error(err))"],"in_app":true}]},"current":true}],"user":{},"modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.30.0","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/gameproj":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"timestamp":"2025-10-04T02:33:58.30438+03:00"}
`)

func nowTime() time.Time {
	return testTime
}

func TestServer_HandleEventIngestion(t *testing.T) {
	t.Parallel()

	t.Run("event ingestion from zapsentry", func(t *testing.T) {
		t.Parallel()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB := testClickHouseDatabaseInstance.NewDatabase(t)

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		projectStore := mysql.NewProjectStore(testDB)
		issueStore := mysql.NewIssueStore(testDB)
		memoryCache := cache.New(5*time.Minute, 10*time.Minute)
		olap := ch.NewClickhouseStore(testOlapDB, svcotel.NewNoopProvider())

		err := projectStore.CreateProject(t.Context(), &warnly.Project{
			CreatedAt: nowTime(),
			Name:      "go-test",
			Key:       "urzovxt",
			UserID:    1,
			TeamID:    1,
			Platform:  warnly.PlatformGolang,
		})
		require.NoError(t, err)

		svc := event.NewEventService(
			projectStore,
			issueStore,
			memoryCache,
			olap,
			nowTime,
		)
		eventHandler := server.NewEventAPIHandler(svc, logger)

		r := httptest.NewRequest(
			http.MethodPost,
			ingestEventPath,
			bytes.NewReader(body))
		r.SetPathValue("project_id", "1")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=urzovxt")

		w := httptest.NewRecorder()

		eventHandler.IngestEvent(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"id":"3708a788c39c44508a3c9442214b2f9f"}`, w.Body.String())
	})

	t.Run("event ingestion with invalid project key", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB := testClickHouseDatabaseInstance.NewDatabase(t)

		projectStore := mysql.NewProjectStore(testDB)
		issueStore := mysql.NewIssueStore(testDB)
		memoryCache := cache.New(5*time.Minute, 10*time.Minute)
		olap := ch.NewClickhouseStore(testOlapDB, svcotel.NewNoopProvider())

		err := projectStore.CreateProject(t.Context(), &warnly.Project{
			CreatedAt: nowTime(),
			Name:      "go-test",
			Key:       "urzovxt",
			UserID:    1,
			TeamID:    1,
			Platform:  warnly.PlatformGolang,
		})
		require.NoError(t, err)

		svc := event.NewEventService(
			projectStore,
			issueStore,
			memoryCache,
			olap,
			nowTime,
		)
		eventHandler := server.NewEventAPIHandler(svc, logger)

		r := httptest.NewRequest(
			http.MethodPost,
			ingestEventPath,
			bytes.NewReader(body))
		r.SetPathValue("project_id", "1")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=invalidkey")

		w := httptest.NewRecorder()

		eventHandler.IngestEvent(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"detail":"project not found","causes":["invalid project identifier or key"]}`, w.Body.String())
	})

	t.Run("event ingestion with error problem from event service", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		svc := NewTestEventService(assert.AnError)
		eventHandler := server.NewEventAPIHandler(svc, logger)

		r := httptest.NewRequest(
			http.MethodPost,
			ingestEventPath,
			bytes.NewReader(body))
		r.SetPathValue("project_id", "1")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.go/0.30.0, sentry_key=urzovxt")

		w := httptest.NewRecorder()

		eventHandler.IngestEvent(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp struct {
			Detail string `json:"detail"`
			//nolint:tagliatelle // keep ErrorID as is for backward compatibility
			ErrorID string `json:"errorId"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Internal Error", resp.Detail)
		require.NoError(t, warnly.ValidateNanoID("errorId", resp.ErrorID))
	})
}

type testEventService struct {
	err error
}

func NewTestEventService(err error) *testEventService {
	return &testEventService{err: err}
}

func (s *testEventService) IngestEvent(ctx context.Context, req warnly.IngestRequest) (warnly.IngestEventResult, error) {
	return warnly.IngestEventResult{}, s.err
}

func TestIngestErrors(t *testing.T) {
	t.Parallel()

	t.Run("NewBadRequestError with wrapped error and causes", func(t *testing.T) {
		t.Parallel()
		origErr := assert.AnError
		err := server.NewBadRequestError("bad request", origErr, "cause1", "cause2")
		assert.Equal(t, http.StatusBadRequest, err.Status)
		assert.Equal(t, "bad request", err.Detail)
		assert.Equal(t, []string{"cause1", "cause2"}, err.Causes)
		assert.Equal(t, origErr, err.WrappedError)
		assert.Contains(t, err.Error(), "bad request (wrapped: assert.AnError general error for testing)")
	})

	t.Run("NewBadRequestError without wrapped error", func(t *testing.T) {
		t.Parallel()
		err := server.NewBadRequestError("bad request", nil, "cause1")
		assert.Equal(t, "bad request: cause1", err.Error())
	})

	t.Run("NewBadRequestError without causes", func(t *testing.T) {
		t.Parallel()
		err := server.NewBadRequestError("bad request", nil)
		assert.Equal(t, "bad request", err.Error())
	})

	t.Run("NewSizeLimitError", func(t *testing.T) {
		t.Parallel()
		err := server.NewSizeLimitError("too large")
		assert.Equal(t, http.StatusBadRequest, err.Status)
		assert.Equal(t, "envelope exceeded size limits", err.Detail)
		assert.Equal(t, []string{"too large"}, err.Causes)
		require.NoError(t, err.WrappedError)
		assert.Equal(t, "envelope exceeded size limits: too large", err.Error())
	})

	t.Run("NewInvalidDSNError", func(t *testing.T) {
		t.Parallel()
		err := server.NewInvalidDSNError()
		assert.Equal(t, http.StatusBadRequest, err.Status)
		assert.Equal(t, "invalid DSN or project key.", err.Detail)
		assert.Empty(t, err.Causes)
		require.NoError(t, err.WrappedError)
		assert.Equal(t, "invalid DSN or project key.", err.Error())
	})

	t.Run("IngestError Unwrap returns wrapped error", func(t *testing.T) {
		t.Parallel()
		origErr := assert.AnError
		err := server.NewBadRequestError("bad request", origErr, "cause1")
		unwrapped := err.Unwrap()
		assert.Equal(t, origErr, unwrapped)
	})

	t.Run("IngestError Unwrap returns nil when no wrapped error", func(t *testing.T) {
		t.Parallel()
		err := server.NewBadRequestError("bad request", nil, "cause1")
		unwrapped := err.Unwrap()
		require.NoError(t, unwrapped)
	})
}
