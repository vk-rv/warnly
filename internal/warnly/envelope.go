package warnly

import (
	"crypto/md5" //nolint:gosec // Non-crypto use
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Uint8Array is a custom type that serializes []uint8 as a JSON array of numbers
// instead of the default base64 encoding for []byte.
type Uint8Array []uint8

func (a Uint8Array) MarshalJSON() ([]byte, error) {
	arr := make([]int, len(a))
	for i, v := range a {
		arr[i] = int(v)
	}
	return json.Marshal(arr)
}

func (a *Uint8Array) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*a = make(Uint8Array, len(arr))
	for i, v := range arr {
		(*a)[i] = uint8(v)
	}
	return nil
}

const defaultExceptionType = "Error"

var ignoredModules = map[string]struct{}{
	"github.com/buger/jsonparser":    {},
	"github.com/rs/zerolog":          {},
	"github.com/getsentry/sentry-go": {},
}

// Event represents the main event structure.
type (
	Event struct {
		EventID string    `json:"event_id"`
		SentAt  time.Time `json:"sent_at"`
		DSN     string    `json:"dsn"`
		SDK     SDK       `json:"sdk"`
		Trace   Trace     `json:"trace"`
	}
	SDK struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	Trace struct {
		Environment string `json:"environment"`
		PublicKey   string `json:"public_key"`
		Release     string `json:"release"`
		TraceID     string `json:"trace_id"`
	}
)

// Contexts represents the various context details.
type Contexts struct {
	OS      OSContext      `json:"os"`
	Device  DeviceContext  `json:"device"`
	Runtime RuntimeContext `json:"runtime"`
}

// DeviceContext represents device-specific information.
type DeviceContext struct {
	Arch   string `json:"arch"`
	NumCPU int    `json:"num_cpu"`
}

// OSContext represents operating system information.
type OSContext struct {
	Name string `json:"name"`
}

// RuntimeContext represents runtime-specific information.
type RuntimeContext struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	GoMaxProcs    int    `json:"go_maxprocs"`
	GoNumCgoCalls int    `json:"go_numcgocalls"`
	GoNumRoutines int    `json:"go_numroutines"`
}

// SDKBody represents SDK details.
type SDKBody struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Integrations []string     `json:"integrations"`
	Packages     []SDKPackage `json:"packages"`
}

// SDKPackage represents individual SDK packages.
type SDKPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Frame represents a single frame in the stack trace.
type Frame struct {
	Function    string   `json:"function"`
	Module      string   `json:"module"`
	AbsPath     string   `json:"abs_path"`
	ContextLine string   `json:"context_line"`
	PreContext  []string `json:"pre_context"`
	PostContext []string `json:"post_context"`
	LineNo      uint32   `json:"lineno"`
	InApp       bool     `json:"in_app"`
}

// StackTrace represents the stack trace details.
type StackTrace struct {
	Frames []Frame `json:"frames"`
}

// Exception represents an individual exception.
type Exception struct {
	Type       string     `json:"type"`
	Value      string     `json:"value"`
	StackTrace StackTrace `json:"stacktrace"`
}

// EventBody represents the main event structure.
// This is the structure that is sent to the Warnly server.
type EventBody struct {
	Timestamp   time.Time         `json:"timestamp"`
	Modules     map[string]string `json:"modules"`
	Tags        map[string]string `json:"tags"`
	User        EventUser         `json:"user"`
	Message     string            `json:"message"`
	Platform    string            `json:"platform"`
	Release     string            `json:"release"`
	ServerName  string            `json:"server_name"`
	Level       string            `json:"level"`
	EventID     string            `json:"event_id"`
	Environment string            `json:"environment"`
	SDK         SDKBody           `json:"sdk"`
	Exception   []Exception       `json:"exception"`
	Extra       map[string]any    `json:"extra"`
	Contexts    Contexts          `json:"contexts"`
}

// EventUser represents user information associated with an event.
type EventUser struct {
	Data      map[string]string `json:"data"`
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	IPAddress string            `json:"ip_address"`
	Username  string            `json:"username"`
	Name      string            `json:"name"`
}

// EventClickhouse represents the event structure for ClickHouse storage.
// It is what ingested into ClickHouse as error events after normalization and processing.
//
//nolint:tagliatelle,tagalign // json tags are used for ClickHouse
type EventClickhouse struct {
	CreatedAt               time.Time  `ch:"created_at" json:"created_at"`
	SDKVersion              string     `ch:"sdk_version" json:"sdk_version"`
	User                    string     `ch:"user" json:"user"`
	UserEmail               string     `ch:"user_email" json:"user_email"`
	UserName                string     `ch:"user_name" json:"user_name"`
	UserUsername            string     `ch:"user_username" json:"user_username"`
	PrimaryHash             string     `ch:"primary_hash" json:"primary_hash"`
	Env                     string     `ch:"env" json:"env"`
	EventID                 string     `ch:"event_id" json:"event_id"`
	Message                 string     `ch:"message" json:"message"`
	IPv6                    string     `ch:"ipv6" json:"ipv6"`
	Release                 string     `ch:"release" json:"release"`
	Title                   string     `ch:"title" json:"title"`
	IPv4                    string     `ch:"ipv4" json:"ipv4"`
	ExceptionFramesInApp    Uint8Array `ch:"exception_frames.in_app" json:"exception_frames.in_app"`
	ContextsKey             []string   `ch:"contexts.key" json:"contexts.key"`
	ExceptionFramesColNo    []uint32   `ch:"exception_frames.colno" json:"exception_frames.colno"`
	ExceptionFramesAbsPath  []string   `ch:"exception_frames.abs_path" json:"exception_frames.abs_path"`
	ExceptionFramesLineNo   []uint32   `ch:"exception_frames.lineno" json:"exception_frames.lineno"`
	ExceptionStacksType     []string   `ch:"exception_stacks.type" json:"exception_stacks.type"`
	ExceptionStacksValue    []string   `ch:"exception_stacks.value" json:"exception_stacks.value"`
	TagsKey                 []string   `ch:"tags.key" json:"tags.key"`
	ExceptionFramesFunction []string   `ch:"exception_frames.function" json:"exception_frames.function"`
	TagsValue               []string   `ch:"tags.value" json:"tags.value"`
	ExceptionFramesFilename []string   `ch:"exception_frames.filename" json:"exception_frames.filename"`
	ContextsValue           []string   `ch:"contexts.value" json:"contexts.value"`
	GroupID                 uint64     `ch:"gid" json:"gid"`
	ProjectID               uint16     `ch:"pid" json:"pid"`
	Level                   uint8      `ch:"level" json:"level"`
	Type                    uint8      `ch:"type" json:"type"`
	SDKID                   uint8      `ch:"sdk_id" json:"sdk_id"`
	Platform                uint8      `ch:"platform" json:"platform"`
	RetentionDays           uint8      `ch:"retention_days" json:"retention_days"`
	Deleted                 uint8      `ch:"deleted" json:"deleted"`
}

// GetExceptionStackTypes returns a list of exception stack types.
func GetExceptionStackTypes(exceptions []Exception) []string {
	if len(exceptions) == 0 {
		return []string{}
	}

	types := make([]string, 0, len(exceptions))
	for i := range exceptions {
		types = append(types, exceptions[i].Type)
	}

	return types
}

// GetExceptionStackValues returns a list of exception stack values.
func GetExceptionStackValues(exceptions []Exception) []string {
	if len(exceptions) == 0 {
		return []string{}
	}

	values := make([]string, 0, len(exceptions))
	for i := range exceptions {
		values = append(values, exceptions[i].Value)
	}

	return values
}

// GetExceptionFramesAbsPath is a exception_frames.abs_path.
func GetExceptionFramesAbsPath(exceptions []Exception) []string {
	if len(exceptions) == 0 {
		return []string{}
	}

	absPath := make([]string, 0, len(exceptions))
	for i := range exceptions {
		for j := range exceptions[i].StackTrace.Frames {
			absPath = append(absPath, exceptions[i].StackTrace.Frames[j].AbsPath)
		}
	}

	return absPath
}

func GetBreaker(exceptions []Exception) string {
	if len(exceptions) == 0 {
		return ""
	}
	e := exceptions[len(exceptions)-1]
	if len(e.StackTrace.Frames) == 0 {
		return ""
	}

	frames := e.StackTrace.Frames

	for i := len(frames) - 1; i >= 0; i-- {
		frame := frames[i]
		if _, found := ignoredModules[frame.Module]; found {
			continue
		}
		return frame.Module + " in " + frame.Function
	}

	firstFrame := frames[0]

	return firstFrame.Module + " in " + firstFrame.Function
}

func GetExceptionValue(exceptions []Exception, defaultVal string) string {
	if len(exceptions) == 0 {
		return defaultVal
	}
	return exceptions[len(exceptions)-1].Value
}

func GetExceptionType(exceptions []Exception, defaultVal string) string {
	if len(exceptions) == 0 {
		return defaultVal
	}
	exc := exceptions[len(exceptions)-1]
	if exc.Type != "" {
		return exc.Type
	}
	if exc.Value != "" {
		return defaultExceptionType
	}

	return defaultVal
}

func GetExceptionFramesColNo(exceptions []Exception) []uint32 {
	if len(exceptions) == 0 {
		return []uint32{}
	}

	colNo := make([]uint32, 0, len(exceptions))
	for i := range exceptions {
		for j := range exceptions[i].StackTrace.Frames {
			colNo = append(colNo, exceptions[i].StackTrace.Frames[j].LineNo)
		}
	}

	return colNo
}

func GetExceptionFramesFilename(exceptions []Exception) []string {
	if len(exceptions) == 0 {
		return []string{}
	}

	filename := make([]string, 0, len(exceptions))
	for i := range exceptions {
		for j := range exceptions[i].StackTrace.Frames {
			name := exceptions[i].StackTrace.Frames[j].AbsPath
			if strings.Contains(name, "/") {
				name = name[strings.LastIndex(name, "/")+1:]
				filename = append(filename, name)
			}
		}
	}

	return filename
}

func GetExceptionFramesFunction(exceptions []Exception) []string {
	if len(exceptions) == 0 {
		return []string{}
	}

	function := make([]string, 0, len(exceptions))
	for i := range exceptions {
		for j := range exceptions[i].StackTrace.Frames {
			function = append(function, exceptions[i].StackTrace.Frames[j].Function)
		}
	}

	return function
}

func GetExceptionFramesLineNo(exceptions []Exception) []uint32 {
	if len(exceptions) == 0 {
		return []uint32{}
	}

	lineNo := make([]uint32, 0, len(exceptions))
	for i := range exceptions {
		for j := range exceptions[i].StackTrace.Frames {
			lineNo = append(lineNo, exceptions[i].StackTrace.Frames[j].LineNo)
		}
	}

	return lineNo
}

func GetExceptionFramesInApp(exceptions []Exception) Uint8Array {
	if len(exceptions) == 0 {
		return Uint8Array{}
	}

	inApp := make(Uint8Array, 0, len(exceptions))
	for i := range exceptions {
		for range exceptions[i].StackTrace.Frames {
			inApp = append(inApp, 1) // for now, all frames are in app
		}
	}

	return inApp
}

func GetHash(event *EventBody) (string, error) {
	if len(event.Exception) > 0 && len(event.Exception[0].StackTrace.Frames) > 0 {
		return GetHashByStackTrace(event)
	}
	return GetHashByMessage(event)
}

func GetHashByStackTrace(event *EventBody) (string, error) {
	h := md5.New() //nolint:gosec // Non-crypto use
	for i := range event.Exception {
		for j := range event.Exception[i].StackTrace.Frames {
			if _, err := h.Write([]byte(event.Exception[i].StackTrace.Frames[j].Module)); err != nil {
				return "", fmt.Errorf("md5: write module: %w", err)
			}
			if _, err := h.Write([]byte(event.Exception[i].StackTrace.Frames[j].Function)); err != nil {
				return "", fmt.Errorf("md5: write function: %w", err)
			}
		}
		if _, err := h.Write([]byte(event.Exception[i].Type)); err != nil {
			return "", fmt.Errorf("md5: write type: %w", err)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetHashByMessage(event *EventBody) (string, error) {
	h := md5.New() //nolint:gosec // Non-crypto use
	if _, err := h.Write([]byte(event.Message)); err != nil {
		return "", fmt.Errorf("md5: write message: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

const (
	HTTPMethodUndefined uint8 = 10
)

// Level represents the event level.
type Level = uint8

const (
	LevelFatal Level = iota + 1
	LevelError
	LevelWarning
	LevelInfo
	LevelDebug
	LevelTrace
	LevelUnknown
)

var levelMapping = map[string]Level{
	"fatal":   LevelFatal,
	"error":   LevelError,
	"warning": LevelWarning,
	"info":    LevelInfo,
	"debug":   LevelDebug,
	"trace":   LevelTrace,
	"unknown": LevelUnknown,
}

// GetLevel returns the level by name.
func GetLevel(level string) Level {
	level = strings.ToLower(level)
	if v, ok := levelMapping[level]; ok {
		return v
	}

	return levelMapping["unknown"]
}
