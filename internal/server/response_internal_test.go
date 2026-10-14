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
		errorResponse{}, loginResponse{}, alertFormResponse{}, settingsResponse{},
		warnly.User{}, warnly.ListProjectsResult{}, warnly.ListIssuesResult{},
		warnly.ProjectDetails{}, warnly.IssueDetails{}, warnly.ListEventsResult{},
		warnly.ListFieldsResult{}, warnly.Discussion{}, warnly.ListAlertsResult{},
		warnly.AnalyticsStoreErr{}, warnly.Schema{}, warnly.SQLQuery{},
		warnly.ProjectInfo{}, warnly.Team{}, warnly.TagCount{}, warnly.TagValueCount{},
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
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			check(typ.Elem())
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				if !field.IsExported() {
					continue
				}
				name := strings.Split(field.Tag.Get("json"), ",")[0]
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
		name   string
		value  any
		status int
		json   string
	}{
		{"login", loginResponse{AuthURL: "/oidc", ProviderName: "keycloak", IsDemo: true}, 200, `{"auth_url":"/oidc","provider_name":"keycloak","is_demo":true}`},
		{"error", errorResponse{Error: "Not found"}, 404, `{"error":"Not found"}`},
		{"new alert", alertFormResponse{Projects: []warnly.Project{}}, 200, `{"projects":[]}`},
		{"encoding failure", make(chan int), 500, `{"error":"Internal Server Error"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tc.status, tc.value)
			require.Equal(t, tc.status, w.Code)
			require.JSONEq(t, tc.json, w.Body.String())
			require.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}
