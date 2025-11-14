package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/server"
	"github.com/vk-rv/warnly/internal/svc/event"
	"github.com/vk-rv/warnly/internal/svc/project"
	"github.com/vk-rv/warnly/internal/warnly"
)

var zerologWithoutErrEvent = []byte(`{"event_id":"d1a2f49ae8614c22b8d0d660517f41ad","sent_at":"2025-10-11T22:52:48.245522+03:00","dsn":"http://urzovxt@0.0.0.0:8030/ingest/2","sdk":{"name":"sentry.go.zerolog","version":"0.35.3"},"trace":{"public_key":"urzovxt","release":"177aa93-dirty","trace_id":"49f652c2b05dd3a26de873a77f5c7136"}}
{"type":"event","length":2837}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":7,"name":"go","version":"go1.24.1"},"trace":{"span_id":"e0a4b0f5e898d023","trace_id":"49f652c2b05dd3a26de873a77f5c7136"}},"event_id":"d1a2f49ae8614c22b8d0d660517f41ad","extra":{"context":"example","key":"1","tag_key":"bar"},"level":"error","message":"Hor error","platform":"go","release":"177aa93-dirty","sdk":{"name":"sentry.go.zerolog","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","user":{"id":"1","email":"testuser@sentry.io","ip_address":"127.0.0.1","username":"testuser","name":"oleg","data":{"foo":"bar","key":"value"}},"logger":"zerolog","modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"timestamp":"2025-10-11T22:52:48.243807+03:00"}
`)

var zerologErrEvent = []byte(`{"event_id":"62fe54af6ffc460eb57b92c67c7c283d","sent_at":"2025-10-11T19:45:14.54031+03:00","dsn":"http://urzovxt@0.0.0.0:8030/ingest/2","sdk":{"name":"sentry.go.zerolog","version":"0.35.3"},"trace":{"public_key":"urzovxt","release":"177aa93-dirty","trace_id":"756c97a71a6a154542cbab25d23e8f2d"}}
{"type":"event","length":6647}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":5,"name":"go","version":"go1.24.1"},"trace":{"span_id":"e893d6300f55bab8","trace_id":"756c97a71a6a154542cbab25d23e8f2d"}},"event_id":"62fe54af6ffc460eb57b92c67c7c283d","extra":{"context":"example","key":"1"},"level":"error","message":"An example error occurred","platform":"go","release":"177aa93-dirty","sdk":{"name":"sentry.go.zerolog","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","user":{"id":"1","email":"testuser@sentry.io","ip_address":"127.0.0.1","username":"testuser","name":"oleg","data":{"foo":"bar","key":"value"}},"logger":"zerolog","modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"exception":[{"value":"an example error occurred at 2025-10-11T19:45:14+03:00","stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":188,"pre_context":["\t\t\tInt8(\"key\", 1).","\t\t\tInterface(\"user\", sentry.User{ID: \"1\", Email: \"testuser@sentry.io\", IPAddress: \"127.0.0.1\", Username: \"testuser\", Name: \"oleg\", Data: map[string]string{","\t\t\t\t\"key\": \"value\",","\t\t\t\t\"foo\": \"bar\",","\t\t\t}})."],"context_line":"\t\t\tMsg(\"An example error occurred\")","post_context":["\t}","","\tterm := make(chan os.Signal, 1)","\tsignal.Notify(term, os.Interrupt)","\ttermCtx, cancel := context.WithCancel(context.Background())"],"in_app":true},{"function":"(*Event).Msg","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":110,"pre_context":["// Calling Msg twice can have unexpected result.","func (e *Event) Msg(msg string) {","\tif e == nil {","\t\treturn","\t}"],"context_line":"\te.msg(msg)","post_context":["}","","// Send is equivalent to calling Msg(\"\").","//","// NOTICE: once this method is called, the *Event should be disposed."],"in_app":true},{"function":"(*Event).msg","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":151,"pre_context":["\t\te.buf = enc.AppendString(enc.AppendKey(e.buf, MessageFieldName), msg)","\t}","\tif e.done != nil {","\t\tdefer e.done(msg)","\t}"],"context_line":"\tif err := e.write(); err != nil {","post_context":["\t\tif ErrorHandler != nil {","\t\t\tErrorHandler(err)","\t\t} else {","\t\t\tfmt.Fprintf(os.Stderr, \"zerolog: could not write event: %v\\n\", err)","\t\t}"],"in_app":true},{"function":"(*Event).write","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/event.go","lineno":80,"pre_context":["\t}","\tif e.level != Disabled {","\t\te.buf = enc.AppendEndMarker(e.buf)","\t\te.buf = enc.AppendLineBreak(e.buf)","\t\tif e.w != nil {"],"context_line":"\t\t\t_, err = e.w.WriteLevel(e.level, e.buf)","post_context":["\t\t}","\t}","\tputEvent(e)","\treturn","}"],"in_app":true},{"function":"multiLevelWriter.WriteLevel","module":"github.com/rs/zerolog","abs_path":"/Users/johndoe/go/pkg/mod/github.com/rs/zerolog@v1.33.0/writer.go","lineno":98,"pre_context":["\treturn n, err","}","","func (t multiLevelWriter) WriteLevel(l Level, p []byte) (n int, err error) {","\tfor _, w := range t.writers {"],"context_line":"\t\tif _n, _err := w.WriteLevel(l, p); err == nil {","post_context":["\t\t\tn = _n","\t\t\tif _err != nil {","\t\t\t\terr = _err","\t\t\t} else if _n != len(p) {","\t\t\t\terr = io.ErrShortWrite"],"in_app":true},{"function":"ObjectEach","module":"github.com/buger/jsonparser","abs_path":"/Users/johndoe/go/pkg/mod/github.com/buger/jsonparser@v1.1.1/parser.go","lineno":1128,"pre_context":["\t\t}","","\t\t// Step 3: find the associated value, then invoke the callback","\t\tif value, valueType, off, err := Get(data[offset:]); err != nil {","\t\t\treturn err"],"context_line":"\t\t} else if err := callback(key, value, valueType, offset+off); err != nil { // Invoke the callback here!","post_context":["\t\t\treturn err","\t\t} else {","\t\t\toffset += off","\t\t}",""],"in_app":true}]}}],"timestamp":"2025-10-11T19:45:14.535877+03:00"}
`)

var zapsentryEventWithoutErr = []byte(`{"event_id":"0018ce1ba9d34f688814c938b74d8e14","sent_at":"2025-10-09T23:59:42.777921+03:00","dsn":"http://urzovxt@127.0.0.1:8030/ingest/2","sdk":{"name":"sentry.go","version":"0.35.3"},"trace":{"environment":"production","public_key":"urzovxt","release":"1.0.0","trace_id":"a23f53eae92c2cf42d940137051165d7"}}
{"type":"event","length":3995}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":5,"name":"go","version":"go1.24.1"},"trace":{"span_id":"9d56580c733cfc96","trace_id":"a23f53eae92c2cf42d940137051165d7"}},"environment":"production","event_id":"0018ce1ba9d34f688814c938b74d8e14","level":"error","message":"My error message at 2025-10-09T23:59:42+03:00","platform":"go","release":"1.0.0","sdk":{"name":"sentry.go","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","threads":[{"stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoe/wasteland/cmd/wasteland/main.go","lineno":170,"pre_context":["","\tfor {","\t\ttime.Sleep(time.Second * 10)","\t\t// randomize error message","\t\tnow := time.Now()"],"context_line":"\t\tmyLogger.Error(fmt.Sprintf(\"My error message at %s\", now.Format(time.RFC3339)), zapsentry.Tag(\"component\", \"my-component\"), zapsentry.Tag(\"key\", \"value\"))","post_context":["\t\t_ = sentryLogger","\t\t//sentryLogger.Error(fmt.Sprintf(\"My info message at %s\", now.Format(time.RFC3339)))","","\t\terr := fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))","\t\t//myLogger.Error(\"An example error occurred\", zap.Error(err), zap.String(\"user_id\", \"12345\"))"],"in_app":true}]},"current":true}],"tags":{"component":"my-component","key":"value"},"user":{},"modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"timestamp":"2025-10-09T23:59:42.775935+03:00"}
`)

var zapsentryEventWithErr = []byte(`{"event_id":"243f84fd26384830b657fe30ea2956bc","sent_at":"2025-10-11T02:59:21.675039+03:00","dsn":"http://urzovxt@127.0.0.1:8030/ingest/2","sdk":{"name":"sentry.go","version":"0.35.3"},"trace":{"environment":"production","public_key":"urzovxt","release":"1.0.0","trace_id":"3181480b4def35afd1fc3d3dbc3e0f81"}}
{"type":"event","length":4184}
{"contexts":{"device":{"arch":"arm64","num_cpu":8},"os":{"name":"darwin"},"runtime":{"go_maxprocs":8,"go_numcgocalls":0,"go_numroutines":6,"name":"go","version":"go1.24.1"},"trace":{"span_id":"45a6199b001477cc","trace_id":"3181480b4def35afd1fc3d3dbc3e0f81"}},"environment":"production","event_id":"243f84fd26384830b657fe30ea2956bc","extra":{"error":"an example error occurred at 2025-10-11T02:59:21+03:00"},"level":"error","message":"my error log message","platform":"go","release":"1.0.0","sdk":{"name":"sentry.go","version":"0.35.3","integrations":["ContextifyFrames","Environment","GlobalTags","IgnoreErrors","IgnoreTransactions","Modules"],"packages":[{"name":"sentry-go","version":"0.35.3"}]},"server_name":"Olegs-MacBook-Pro.local","tags":{"component":"my-component","key":"value"},"user":{},"modules":{"":"","github.com/BurntSushi/toml":"v1.2.1","github.com/TheZeroSlave/zapsentry":"v1.23.0","github.com/benbjohnson/agency":"v0.0.0-20170601160516-33de8fbf97c4","github.com/beorn7/perks":"v1.0.1","github.com/buger/jsonparser":"v1.1.1","github.com/cespare/xxhash/v2":"v2.2.0","github.com/dgryski/go-rendezvous":"v0.0.0-20200823014737-9f7001d12a5f","github.com/getsentry/sentry-go":"v0.35.3","github.com/getsentry/sentry-go/zerolog":"v0.35.3","github.com/go-chi/chi":"v1.5.4","github.com/go-sql-driver/mysql":"v1.7.1","github.com/golang/protobuf":"v1.5.3","github.com/ilyakaznacheev/cleanenv":"v1.5.0","github.com/joho/godotenv":"v1.5.1","github.com/josharian/intern":"v1.0.0","github.com/mailru/easyjson":"v0.7.7","github.com/matoous/go-nanoid/v2":"v2.0.0","github.com/mattn/go-colorable":"v0.1.13","github.com/mattn/go-isatty":"v0.0.20","github.com/matttproud/golang_protobuf_extensions":"v1.0.4","github.com/mroth/weightedrand/v2":"v2.1.0","github.com/newrelic/go-agent/v3":"v3.24.1","github.com/newrelic/go-agent/v3/integrations/nrredis-v9":"v1.0.0","github.com/prometheus/client_golang":"v1.16.0","github.com/prometheus/client_model":"v0.3.0","github.com/prometheus/common":"v0.42.0","github.com/prometheus/procfs":"v0.10.1","github.com/redis/go-redis/v9":"v9.0.5","github.com/rs/zerolog":"v1.33.0","github.com/shopspring/decimal":"v1.3.1","github.com/sourcegraph/conc":"v0.3.0","github.com/valyala/bytebufferpool":"v1.0.0","github.com/valyala/quicktemplate":"v1.7.0","github.com/vk-rv/wasteland":"(devel)","go.uber.org/multierr":"v1.10.0","go.uber.org/zap":"v1.25.0","golang.org/x/crypto":"v0.21.0","golang.org/x/net":"v0.23.0","golang.org/x/sys":"v0.18.0","golang.org/x/text":"v0.14.0","google.golang.org/genproto":"v0.0.0-20230110181048-76db0878b65f","google.golang.org/grpc":"v1.54.0","google.golang.org/protobuf":"v1.33.0","gopkg.in/yaml.v3":"v3.0.1","olympos.io/encoding/edn":"v0.0.0-20201019073823-d3554ca0b0a3"},"exception":[{"type":"*errors.errorString","value":"an example error occurred at 2025-10-11T02:59:21+03:00","stacktrace":{"frames":[{"function":"main","module":"main","abs_path":"/Users/johndoev/wasteland/cmd/wasteland/main.go","lineno":98,"pre_context":["\tif err != nil {","\t\tfmt.Printf(\"failed to create logger: %s\\n\", err)","\t\tos.Exit(failed)","\t}",""],"context_line":"\tif err := run(logger, atomicLevel); err != nil {","post_context":["\t\tlogger.Error(\"wasteland web server start / shutdown problem\", zap.Error(err))","\t\tos.Exit(failed)","\t}","}",""],"in_app":true},{"function":"run","module":"main","abs_path":"/Users/johndoev/wasteland/cmd/wasteland/main.go","lineno":171,"pre_context":["\tfor {","\t\ttime.Sleep(time.Second * 10)","\t\t// randomize error message","\t\tnow := time.Now()","\t\terr := fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))"],"context_line":"\t\tmyLogger.Error(\"my error log message\", zap.Error(err), zapsentry.Tag(\"component\", \"my-component\"), zapsentry.Tag(\"key\", \"value\"))","post_context":["\t\t_ = sentryLogger","\t\t//sentryLogger.Error(fmt.Sprintf(\"My info message at %s\", now.Format(time.RFC3339)))","","\t\terr = fmt.Errorf(\"an example error occurred at %s\", now.Format(time.RFC3339))","\t\t//myLogger.Error(\"An example error occurred\", zap.Error(err), zap.String(\"user_id\", \"12345\"))"],"in_app":true}]}}],"timestamp":"2025-10-11T02:59:21.674005+03:00"}`)

const (
	issTypeClass           = ".iss-type"
	issMsgClass            = ".iss-msg"
	issViewClass           = ".iss-view"
	projectErrorCountClass = ".project-error-count"
	projectCardClass       = ".project-card"
)

func generateUniqueEventPayload(basePayload []byte, index int) []byte {
	payload := string(basePayload)

	newEventID := strings.ReplaceAll(uuid.New().String(), "-", "")
	newTraceID := strings.ReplaceAll(uuid.New().String(), "-", "")
	newSpanID := strings.ReplaceAll(uuid.New().String(), "-", "")[:16]

	oldEventID := "243f84fd26384830b657fe30ea2956bc"
	payload = strings.ReplaceAll(payload, oldEventID, newEventID)

	oldTraceID := "3181480b4def35afd1fc3d3dbc3e0f81"
	payload = strings.ReplaceAll(payload, oldTraceID, newTraceID)

	oldSpanID := "45a6199b001477cc"
	payload = strings.ReplaceAll(payload, oldSpanID, newSpanID)

	oldSentAt := "2025-10-11T02:59:21.675039+03:00"
	newSentAt := fmt.Sprintf("2025-10-11T02:59:%02d.675039+03:00", 21+index)
	payload = strings.ReplaceAll(payload, oldSentAt, newSentAt)

	oldTimestamp := "2025-10-11T02:59:21.674005+03:00"
	newTimestamp := fmt.Sprintf("2025-10-11T02:59:%02d.674005+03:00", 21+index)
	payload = strings.ReplaceAll(payload, oldTimestamp, newTimestamp)

	oldMsg := "my error log message"
	newMsg := fmt.Sprintf("my error log message %d", index)
	payload = strings.ReplaceAll(payload, oldMsg, newMsg)

	return []byte(payload)
}

var (
	zapsentryErrExpectedType = "*errors.errorString"
	zapsentryErrExpectedView = "main in run"
	zapsentryErrExpectedMsg  = "an example error occurred at 2025-10-11T02:59:21+03:00"

	zapsentryNoErrExpectedType = "My error message at 2025-10-09T23:59:42+03:00"
	zapsentryNoErrExpectedMsg  = "(No error message)"

	zerologErrExpectedType = "Error"
	zerologErrExpectedView = "main in run"
	zerologErrExpectedMsg  = "an example error occurred at 2025-10-11T19:45:14+03:00"

	zerologNoErrExpectedType = "Hor error"
	zerologNoErrExpectedMsg  = "(No error message)"
	zerologNoErrExpectedView = ""
)

func TestServer_ListProjects(t *testing.T) {
	t.Parallel()

	t.Run("list all projects", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, _ := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

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

		for i := 1; i <= 3; i++ {
			require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
				CreatedAt: nowTime(),
				Name:      fmt.Sprintf("project-%d", i),
				Key:       fmt.Sprintf("key-%d", i),
				UserID:    testOwnerID,
				TeamID:    testOwnerID,
				Platform:  warnly.PlatformGolang,
			}))
		}

		w, r := getListProjectsRequest(ctx, "", 0)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 3, projectCards.Length(), "should have exactly 3 projects")
	})

	t.Run("list projects filtered by team", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, _ := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

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
			testBaseURL,
			testBaseScheme,
			nowTime,
			logger,
		)
		projectHandler := server.NewProjectHandler(projectSvc, logger)

		require.NoError(t, s.teamStore.CreateTeam(ctx, warnly.Team{
			CreatedAt: nowTime(),
			Name:      "team-1",
			OwnerID:   testOwnerID,
		}))
		require.NoError(t, s.teamStore.CreateTeam(ctx, warnly.Team{
			CreatedAt: nowTime(),
			Name:      "team-2",
			OwnerID:   testOwnerID,
		}))

		require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
			CreatedAt: nowTime(),
			Name:      "project-team-1",
			Key:       "key1",
			UserID:    testOwnerID,
			TeamID:    1,
			Platform:  warnly.PlatformGolang,
		}))
		require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
			CreatedAt: nowTime(),
			Name:      "project-team-2",
			Key:       "key2",
			UserID:    testOwnerID,
			TeamID:    2,
			Platform:  warnly.PlatformGolang,
		}))

		w, r := getListProjectsRequest(ctx, "", 1)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 1, projectCards.Length(), "should have exactly 1 project filtered by team")
	})

	t.Run("list projects filtered by name", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, _ := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

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
			Name:      "api-service",
			Key:       "api-key",
			UserID:    testOwnerID,
			TeamID:    testOwnerID,
			Platform:  warnly.PlatformGolang,
		}))
		require.NoError(t, s.projectStore.CreateProject(ctx, &warnly.Project{
			CreatedAt: nowTime(),
			Name:      "web-app",
			Key:       "web-key",
			UserID:    testOwnerID,
			TeamID:    testOwnerID,
			Platform:  warnly.PlatformGolang,
		}))

		w, r := getListProjectsRequest(ctx, "api", 0)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 1, projectCards.Length(), "should have exactly 1 project filtered by name")
	})

	t.Run("list projects with both team and name filters", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, _ := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

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
			Name:      "api-service",
			Key:       "api-key",
			UserID:    testOwnerID,
			TeamID:    1,
			Platform:  warnly.PlatformGolang,
		}))

		w, r := getListProjectsRequest(ctx, "api", 1)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 1, projectCards.Length(), "should have exactly 1 project with both filters")
	})

	t.Run("list projects with empty result", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, _ := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

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

		w, r := getListProjectsRequest(ctx, "nonexistent", 0)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 0, projectCards.Length(), "should have no projects for nonexistent search")
	})

	t.Run("list projects with event ingestion and event count verification", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
		testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
		logger, buf := getTestLogger()
		s := getTestStores(testDB, testOlapDB, logger)

		eventSvc := event.NewEventService(
			s.projectStore,
			s.issueStore,
			s.memoryCache,
			s.olap,
			nowHalfAnHourBefore,
		)
		eventHandler := server.NewEventAPIHandler(eventSvc, logger)

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
			Name:      "project-with-events",
			Key:       testProjectKey,
			UserID:    testOwnerID,
			TeamID:    testOwnerID,
			Platform:  warnly.PlatformGolang,
		}))

		for i := range 3 {
			wIngest, rIngest := getIngestRequest(generateUniqueEventPayload(zapsentryEventWithErr, i))
			eventHandler.IngestEvent(wIngest, rIngest)
			if wIngest.Code != http.StatusOK {
				t.Log("=== LOGGER OUTPUT ===")
				t.Log(buf.String())
				t.Logf("Response body: %s", wIngest.Body.String())
			}
			require.Equal(t, http.StatusOK, wIngest.Code)
		}

		w, r := getListProjectsRequest(ctx, "", 0)
		projectHandler.ListProjects(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		doc, err := goquery.NewDocumentFromReader(w.Body)
		require.NoError(t, err)

		errorCount := doc.Find(projectErrorCountClass).First().Text()
		assert.Equal(t, "3", errorCount)

		projectCards := doc.Find(projectCardClass)
		assert.Equal(t, 1, projectCards.Length(), "should have exactly 1 project")
	})
}

func TestServer_HandleProjectDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		expectedIssueType string
		expectedIssueMsg  string
		expectedIssueView string
		eventPayload      []byte
	}{
		{
			name:              "Go Zerolog: Project issue naming and output in project details without error",
			eventPayload:      zerologWithoutErrEvent,
			expectedIssueType: zerologNoErrExpectedType,
			expectedIssueMsg:  zerologNoErrExpectedMsg,
			expectedIssueView: zerologNoErrExpectedView,
		},
		{
			name:              "Go Zerolog: Project issue naming and output in project details with error",
			eventPayload:      zerologErrEvent,
			expectedIssueType: zerologErrExpectedType,
			expectedIssueMsg:  zerologErrExpectedMsg,
			expectedIssueView: zerologErrExpectedView,
		},
		{
			name:              "Go Zap: Project issue naming and output in project details with error",
			eventPayload:      zapsentryEventWithErr,
			expectedIssueType: zapsentryErrExpectedType,
			expectedIssueMsg:  zapsentryErrExpectedMsg,
			expectedIssueView: zapsentryErrExpectedView,
		},
		{
			name:              "Go Zap: Project issue naming and output in project details without error",
			eventPayload:      zapsentryEventWithoutErr,
			expectedIssueType: zapsentryNoErrExpectedType,
			expectedIssueMsg:  zapsentryNoErrExpectedMsg,
			expectedIssueView: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			testDB, _ := testMySQLDatabaseInstance.NewDatabase(t)
			testOlapDB, _ := testClickHouseDatabaseInstance.NewDatabase(t)
			logger, _ := getTestLogger()
			s := getTestStores(testDB, testOlapDB, logger)

			eventSvc := event.NewEventService(
				s.projectStore,
				s.issueStore,
				s.memoryCache,
				s.olap,
				nowHalfAnHourBefore,
			)
			eventHandler := server.NewEventAPIHandler(eventSvc, logger)

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

			wIngest, rIngest := getIngestRequest(tt.eventPayload)
			eventHandler.IngestEvent(wIngest, rIngest)
			assert.Equal(t, http.StatusOK, wIngest.Code)

			wrDetails, rrDetails := getProjectDetailsRequest(ctx)
			projectHandler.ProjectDetails(wrDetails, rrDetails)
			assert.Equal(t, http.StatusOK, wrDetails.Code)

			doc, err := goquery.NewDocumentFromReader(wrDetails.Body)
			require.NoError(t, err)

			issueType := doc.Find(issTypeClass).First().Text()
			issueView := doc.Find(issViewClass).First().Text()
			issueMsg := doc.Find(issMsgClass).First().Text()

			assert.Equal(t, tt.expectedIssueType, issueType)
			assert.Equal(t, tt.expectedIssueMsg, issueMsg)
			assert.Equal(t, tt.expectedIssueView, issueView)
		})
	}
}
