package server

import (
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

// Walk types rather than zero-value JSON: nil pointers and empty slices must not
// hide untagged fields in nested API responses.
func TestAPIResponseFieldNames(t *testing.T) {
	t.Parallel()
	responses := []any{
		errorResponse{},
		loginResponse{},
		alertFormResponse{},
		settingsResponse{},
		warnly.User{},
		warnly.ListProjectsResult{},
		warnly.ListIssuesResult{},
		warnly.ProjectDetails{},
		warnly.IssueDetails{},
		warnly.ListEventsResult{},
		warnly.ListFieldsResult{},
		warnly.Discussion{},
		warnly.ListAlertsResult{},
		warnly.AnalyticsStoreErr{},
		warnly.Schema{},
		warnly.SQLQuery{},
		warnly.ProjectInfo{},
		warnly.Team{},
		warnly.TagCount{},
		warnly.TagValueCount{},
	}
	valid := regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	seen := make(map[reflect.Type]bool)
	var check func(reflect.Type)
	check = func(typ reflect.Type) {
		if seen[typ] || typ == reflect.TypeFor[time.Time]() {
			return
		}
		seen[typ] = true
		switch typ.Kind() {
		case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan,
			reflect.Func, reflect.Interface, reflect.String, reflect.UnsafePointer:
			return
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			check(typ.Elem())
		case reflect.Struct:
			for field := range typ.Fields() {
				if !field.IsExported() {
					continue
				}
				name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
				if name == "-" {
					continue
				}
				if !field.Anonymous {
					require.Regexp(t, valid, name, "%s.%s requires an explicit snake_case JSON tag", typ, field.Name)
				}
				check(field.Type)
			}
		}
	}
	for _, response := range responses {
		check(reflect.TypeOf(response))
	}
}

func TestAPIResponseEnvelopes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value  any
		name   string
		json   string
		status int
	}{
		{name: "login", value: loginResponse{AuthURL: "/oidc", ProviderName: "keycloak", IsDemo: true}, status: 200, json: `{"auth_url":"/oidc","provider_name":"keycloak","is_demo":true}`},
		{name: "error", value: errorResponse{Error: "Not found"}, status: 404, json: `{"error":"Not found"}`},
		{name: "new alert", value: alertFormResponse{Projects: []warnly.Project{}}, status: 200, json: `{"projects":[]}`},
		{name: "encoding failure", value: make(chan int), status: 500, json: `{"error":"Internal Server Error"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			writeJSON(w, tc.status, tc.value)
			require.Equal(t, tc.status, w.Code)
			require.JSONEq(t, tc.json, w.Body.String())
			require.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}
