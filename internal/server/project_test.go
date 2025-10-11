package server_test

import (
	"net/http"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/server"
	"github.com/vk-rv/warnly/internal/svc/event"
	"github.com/vk-rv/warnly/internal/svc/project"
	"github.com/vk-rv/warnly/internal/warnly"
)

//nolint:unused // reserved for future usage
var zerologEvent = []byte(`{"type":"event","length":6793}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":7,"name":"go","version":"go1.24.1"},"trace":{"span_id":"c75099cab60c74b0","trace_id":"6862bf3d5771f7b28169d386ef177394"}},"event_id":"5da840e72d9547628fbc70835fb16732","extra":{"context":"example","key":"1"},"level":"error","message":"An example error occurred","platform":"go","release":"177aa93-dirty","sdk":{"name":"sentry.go.zerolog","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","user":{"id":"1","email":"testuser@sentry.io","ip_address":"127.0.0.1","username":"testuser","name":"oleg","data":{"foo":"bar","key":"value"}},"logger":"zerolog","modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"exception":[{"value":"an example error occurred at 2025-10-09T23:59:42+03:00","stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":181,"pre_context":["","\t\t//sentryLogger.Error(\"An example info message\", zap.Error(err), zap.String(\"user_id\", \"12345\"))","\t\tzlogger.Error().Err(err).Interface(\"context\", \"example\").Int8(\"key\", 1).Interface(\"user\", sentry.User{ID: \"1\", Email: \"testuser@sentry.io\", IPAddress: \"127.0.0.1\", Username: \"testuser\", Name: \"oleg\", Data: map[string]string{","\t\t\t\"key\": \"value\",","\t\t\t\"foo\": \"bar\","],"context_line":"\t\t}}).Msg(\"An example error occurred\")","post_context":["\t}","","\tterm := make(chan os.Signal, 1)","\tsignal.Notify(term, os.Interrupt)","\ttermCtx, cancel := context.WithCancel(context.Background())"],"in_app":true},{"function":"(*Event).Msg","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":110,"pre_context":["// Calling Msg twice can have unexpected result.","func (e *Event) Msg(msg string) {","\tif e == nil {","\t\treturn","\t}"],"context_line":"\te.msg(msg)","post_context":["}","","// Send is equivalent to calling Msg(\"\").","//","// NOTICE: once this method is called, the *Event should be disposed."],"in_app":true},{"function":"(*Event).msg","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":151,"pre_context":["\t\te.buf = enc.AppendString(enc.AppendKey(e.buf, MessageFieldName), msg)","\t}","\tif e.done != nil {","\t\tdefer e.done(msg)","\t}"],"context_line":"\tif err := e.write(); err != nil {","post_context":["\t\tif ErrorHandler != nil {","\t\t\tErrorHandler(err)","\t\t} else {","\t\t\tfmt.Fprintf(os.Stderr, \"zerolog: could not write event: %v\\n\", err)","\t\t}"],"in_app":true},{"function":"(*Event).write","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":80,"pre_context":["\t}","\tif e.level != Disabled {","\t\te.buf = enc.AppendEndMarker(e.buf)","\t\te.buf = enc.AppendLineBreak(e.buf)","\t\tif e.w != nil {"],"context_line":"\t\t\t_, err = e.w.WriteLevel(e.level, e.buf)","post_context":["\t\t}","\t}","\tputEvent(e)","\treturn","}"],"in_app":true},{"function":"multiLevelWriter.WriteLevel","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/writer.go","lineno":98,"pre_context":["\treturn n, err","}","","func (t multiLevelWriter) WriteLevel(l Level, p []byte) (n int, err error) {","\tfor _, w := range t.writers {"],"context_line":"\t\tif _n, _err := w.WriteLevel(l, p); err == nil {","post_context":["\t\t\tn = _n","\t\t\tif _err != nil {","\t\t\t\terr = _err","\t\t\t} else if _n != len(p) {","\t\t\t\terr = io.ErrShortWrite"],"in_app":true},{"function":"ObjectEach","module":"github.com/buger/jsonparser","abs_path":"/Users/johndoe/go/pkg/mod/github.com/buger/jsonparser@v1.1.1/parser.go","lineno":1128,"pre_context":["\t\t}","","\t\t// Step 3: find the associated value, then invoke the callback","\t\tif value, valueType, off, err := Get(data[offset:]); err != nil {","\t\t\treturn err"],"context_line":"\t\t} else if err := callback(key, value, valueType, offset+off); err != nil { // Invoke the callback here!","post_context":["\t\t\treturn err","\t\t} else {","\t\t\toffset += off","\t\t}",""],"in_app":true}]}}],"timestamp":"2025-10-09T23:59:42.778335+03:00"}
`)

var zapsentryEventWithoutErr = []byte(`{"event_id":"0018ce1ba9d34f688814c938b74d8e14","sent_at":"2025-10-09T23:59:42.777921+03:00","dsn":"http://urzovxt@127.0.0.1:8030/ingest/2","sdk":{"name":"sentry.go","version":"0.35.3"},"trace":{"environment":"production","public_key":"urzovxt","release":"1.0.0","trace_id":"a23f53eae92c2cf42d940137051165d7"}}
{"type":"event","length":3995}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":5,"name":"go","version":"go1.24.1"},"trace":{"span_id":"9d56580c733cfc96","trace_id":"a23f53eae92c2cf42d940137051165d7"}},"environment":"production","event_id":"0018ce1ba9d34f688814c938b74d8e14","level":"error","message":"My error message at 2025-10-09T23:59:42+03:00","platform":"go","release":"1.0.0","sdk":{"name":"sentry.go","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","threads":[{"stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":170,"pre_context":["","\tfor {","\t\ttime.Sleep(time.Second * 10)","\t\t// randomize error message","\t\tnow := time.Now()"],"context_line":"\t\tmyLogger.Error(fmt.Sprintf(\"My error message at %s\", now.Format(time.RFC3339)), zapsentry.Tag(\"component\", \"my-component\"), zapsentry.Tag(\"key\", \"value\"))","post_context":["\t\t_ = sentryLogger","\t\t//sentryLogger.Error(fmt.Sprintf(\"My info message at %s\", now.Format(time.RFC3339)))","","\t\terr := fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))","\t\t//myLogger.Error(\"An example error occurred\", zap.Error(err), zap.String(\"user_id\", \"12345\"))"],"in_app":true}]},"current":true}],"tags":{"component":"my-component","key":"value"},"user":{},"modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"timestamp":"2025-10-09T23:59:42.775935+03:00"}
`)

var zapsentryEventWithErr = []byte(`{"event_id":"243f84fd26384830b657fe30ea2956bc","sent_at":"2025-10-11T02:59:21.675039+03:00","dsn":"http://urzovxt@127.0.0.1:8030/ingest/2","sdk":{"name":"sentry.go","version":"0.35.3"},"trace":{"environment":"production","public_key":"urzovxt","release":"1.0.0","trace_id":"3181480b4def35afd1fc3d3dbc3e0f81"}}
{"type":"event","length":4184}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":6,"name":"go","version":"go1.24.1"},"trace":{"span_id":"45a6199b001477cc","trace_id":"3181480b4def35afd1fc3d3dbc3e0f81"}},"environment":"production","event_id":"243f84fd26384830b657fe30ea2956bc","extra":{"error":"an example error occurred at 2025-10-11T02:59:21+03:00"},"level":"error","message":"my error log message","platform":"go","release":"1.0.0","sdk":{"name":"sentry.go","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","tags":{"component":"my-component","key":"value"},"user":{},"modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"exception":[{"type":"*errors.errorString","value":"an example error occurred at 2025-10-11T02:59:21+03:00","stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoev/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoev/wasteland/cmd/wasteland/main.go","lineno":171,"pre_context":["\tfor {","\t\ttime.Sleep(time.Second * 10)","\t\t// randomize error message","\t\tnow := time.Now()","\t\terr := fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))"],"context_line":"\t\tmyLogger.Error(\"my error log message\", zap.Error(err), zapsentry.Tag(\"component\", \"my-component\"), zapsentry.Tag(\"key\", \"value\"))","post_context":["\t\t_ = sentryLogger","\t\t//sentryLogger.Error(fmt.Sprintf(\"My info message at %s\", now.Format(time.RFC3339)))","","\t\terr = fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))","\t\t//myLogger.Error(\"An example error occurred\", zap.Error(err), zap.String(\"user_id\", \"12345\"))"],"in_app":true}]}}],"timestamp":"2025-10-11T02:59:21.674005+03:00"}`)

const projectDetailsPath = "/projects/{id}?issues=all&period=1h"

const (
	issTypeClass = ".iss-type"
	issMsgClass  = ".iss-msg"
	issViewClass = ".iss-view"
)

var (
	// for zapsentry event with error, obtained from exception[0].type and exception[0].value.
	zapsentryErrExpectedType = "*errors.errorString"
	zapsentryErrExpectedView = "main in run"
	zapsentryErrExpectedMsg  = "an example error occurred at 2025-10-11T02:59:21+03:00"

	// for zapsentry event without error, obtained from message and no exception.
	zapsentryNoErrExpectedType = "My error message at 2025-10-09T23:59:42+03:00"
	zapsentryNoErrExpectedMsg  = "(No error message)"
)

func TestServer_HandleProjectDetails(t *testing.T) {
	t.Parallel()

	t.Run("Go Zap: Project issue naming and output in project details with error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB := testClickHouseDatabaseInstance.NewDatabase(t)

		logger, _ := getTestLogger()

		s := getTestStores(testDB, testOlapDB, logger)

		svc := event.NewEventService(
			s.projectStore,
			s.issueStore,
			s.memoryCache,
			s.olap,
			nowHalfAnHourBefore,
		)
		eventHandler := server.NewEventAPIHandler(svc, logger)

		projectSvc := project.NewProjectService(
			s.projectStore,
			s.assingmentStore,
			s.teamStore,
			s.issueStore,
			s.messageStore,
			s.mentionStore,
			s.olap,
			s.uow,
			bluemonday.NewPolicy(),
			testBaseURL,
			testBaseScheme,
			nowTime,
			logger,
		)

		projectHandler := server.NewProjectHandler(projectSvc, logger)

		require.NoError(t, s.teamStore.CreateTeam(ctx, warnly.Team{
			CreatedAt: nowTime(),
			Name:      testTeamName,
			OwnerID:   testOwnerID,
		}))
		require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
			CreatedAt: nowTime(),
			Name:      testProjectName,
			Key:       testProjectKey,
			UserID:    testOwnerID,
			TeamID:    testOwnerID,
			Platform:  warnly.PlatformGolang,
		}))

		w, r := getIngestRequest(zapsentryEventWithErr)

		eventHandler.IngestEvent(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		wr, rr := getProjectDetailsRequest(ctx)

		projectHandler.ProjectDetails(wr, rr)

		doc, err := goquery.NewDocumentFromReader(wr.Body)
		require.NoError(t, err)

		issueType := doc.Find(issTypeClass).First().Text()
		issueView := doc.Find(issViewClass).First().Text()
		issueMsg := doc.Find(issMsgClass).First().Text()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, zapsentryErrExpectedType, issueType)
		assert.Equal(t, zapsentryErrExpectedMsg, issueMsg)
		assert.Equal(t, zapsentryErrExpectedView, issueView)
	})

	t.Run("Go Zap: Project issue naming and output in project details without error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB := testClickHouseDatabaseInstance.NewDatabase(t)

		logger, _ := getTestLogger()

		s := getTestStores(testDB, testOlapDB, logger)

		svc := event.NewEventService(
			s.projectStore,
			s.issueStore,
			s.memoryCache,
			s.olap,
			nowHalfAnHourBefore,
		)
		eventHandler := server.NewEventAPIHandler(svc, logger)

		projectSvc := project.NewProjectService(
			s.projectStore,
			s.assingmentStore,
			s.teamStore,
			s.issueStore,
			s.messageStore,
			s.mentionStore,
			s.olap,
			s.uow,
			bluemonday.NewPolicy(),
			testBaseURL,
			testBaseScheme,
			nowTime,
			logger,
		)

		projectHandler := server.NewProjectHandler(projectSvc, logger)

		require.NoError(t, s.teamStore.CreateTeam(ctx, warnly.Team{
			CreatedAt: nowTime(),
			Name:      testTeamName,
			OwnerID:   testOwnerID,
		}))
		require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
			CreatedAt: nowTime(),
			Name:      testProjectName,
			Key:       testProjectKey,
			UserID:    testOwnerID,
			TeamID:    testOwnerID,
			Platform:  warnly.PlatformGolang,
		}))

		w, r := getIngestRequest(zapsentryEventWithoutErr)

		eventHandler.IngestEvent(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		wr, rr := getProjectDetailsRequest(ctx)

		projectHandler.ProjectDetails(wr, rr)

		doc, err := goquery.NewDocumentFromReader(wr.Body)
		require.NoError(t, err)

		issueType := doc.Find(issTypeClass).First().Text()
		issueView := doc.Find(issViewClass).First().Text()
		issueMsg := doc.Find(issMsgClass).First().Text()

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, zapsentryNoErrExpectedType, issueType)
		assert.Equal(t, zapsentryNoErrExpectedMsg, issueMsg)
		assert.Empty(t, issueView)
	})
}
