package warnly_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestProjectDetailsAllLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.ProjectDetails
		want    string
	}{
		{
			name: "zero length returns empty string",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 0,
				},
			},
			want: "",
		},
		{
			name: "small number",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 42,
				},
			},
			want: "42",
		},
		{
			name: "number in thousands",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 5000,
				},
			},
			want: "5.0k",
		},
		{
			name: "number in millions",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 1500000,
				},
			},
			want: "1.5m",
		},
		{
			name: "exactly 1000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 1000,
				},
			},
			want: "1000",
		},
		{
			name: "just over 1000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 1001,
				},
			},
			want: "1.0k",
		},
		{
			name: "exactly 1000000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 1000000,
				},
			},
			want: "1000.0k",
		},
		{
			name: "just over 1000000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 1000001,
				},
			},
			want: "1.0m",
		},
		{
			name: "large number",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					AllLength: 999,
				},
			},
			want: "999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.AllLength()
			if result != tt.want {
				t.Errorf("AllLength() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestProjectDetailsNewLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.ProjectDetails
		want    string
	}{
		{
			name: "zero length returns empty string",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 0,
				},
			},
			want: "",
		},
		{
			name: "small number",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 7,
				},
			},
			want: "7",
		},
		{
			name: "number in thousands",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 2500,
				},
			},
			want: "2.5k",
		},
		{
			name: "number in millions",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 3000000,
				},
			},
			want: "3.0m",
		},
		{
			name: "exactly 1000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 1000,
				},
			},
			want: "1000",
		},
		{
			name: "just over 1000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 1001,
				},
			},
			want: "1.0k",
		},
		{
			name: "exactly 1000000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 1000000,
				},
			},
			want: "1000.0k",
		},
		{
			name: "just over 1000000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 1000001,
				},
			},
			want: "1.0m",
		},
		{
			name: "number just under 1000",
			details: &warnly.ProjectDetails{
				Project: &warnly.Project{
					NewLength: 999,
				},
			},
			want: "999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.NewLength()
			if result != tt.want {
				t.Errorf("NewLength() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestFieldValueNumPercentsFormatted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fvn     *warnly.FieldValueNum
		wantStr string
	}{
		{
			name: "whole number",
			fvn: &warnly.FieldValueNum{
				Tag:             "browser",
				Value:           "Chrome",
				PercentsOfTotal: 50,
			},
			wantStr: "50",
		},
		{
			name: "decimal value floors down",
			fvn: &warnly.FieldValueNum{
				Tag:             "browser",
				Value:           "Firefox",
				PercentsOfTotal: 50.9,
			},
			wantStr: "50",
		},
		{
			name: "very small value",
			fvn: &warnly.FieldValueNum{
				Tag:             "device",
				Value:           "unknown",
				PercentsOfTotal: 0.1,
			},
			wantStr: "0",
		},
		{
			name: "zero percent",
			fvn: &warnly.FieldValueNum{
				Tag:             "env",
				Value:           "dev",
				PercentsOfTotal: 0,
			},
			wantStr: "0",
		},
		{
			name: "full 100 percent",
			fvn: &warnly.FieldValueNum{
				Tag:             "status",
				Value:           "success",
				PercentsOfTotal: 100,
			},
			wantStr: "100",
		},
		{
			name: "decimal close to whole number",
			fvn: &warnly.FieldValueNum{
				Tag:             "platform",
				Value:           "iOS",
				PercentsOfTotal: 75.99999,
			},
			wantStr: "75",
		},
		{
			name: "single decimal place",
			fvn: &warnly.FieldValueNum{
				Tag:             "version",
				Value:           "1.0",
				PercentsOfTotal: 33.3,
			},
			wantStr: "33",
		},
		{
			name: "large decimal",
			fvn: &warnly.FieldValueNum{
				Tag:             "error",
				Value:           "timeout",
				PercentsOfTotal: 99.99,
			},
			wantStr: "99",
		},
		{
			name: "mid-range value with decimals",
			fvn: &warnly.FieldValueNum{
				Tag:             "region",
				Value:           "us-west",
				PercentsOfTotal: 42.7,
			},
			wantStr: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.fvn.PercentsFormatted()
			if result != tt.wantStr {
				t.Errorf("PercentsFormatted() = %q, want %q", result, tt.wantStr)
			}
		})
	}
}

func TestListTagValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tag     string
		tv      []warnly.FieldValueNum
		wantLen int
	}{
		{
			name: "tag with multiple values",
			tag:  "browser",
			tv: []warnly.FieldValueNum{
				{Tag: "browser", Value: "Chrome", PercentsOfTotal: 50},
				{Tag: "browser", Value: "Firefox", PercentsOfTotal: 30},
				{Tag: "browser", Value: "Safari", PercentsOfTotal: 20},
				{Tag: "os", Value: "Windows", PercentsOfTotal: 60},
				{Tag: "os", Value: "macOS", PercentsOfTotal: 40},
			},
			wantLen: 3,
		},
		{
			name: "tag with single value",
			tag:  "version",
			tv: []warnly.FieldValueNum{
				{Tag: "version", Value: "1.0.0", PercentsOfTotal: 100},
				{Tag: "environment", Value: "production", PercentsOfTotal: 100},
			},
			wantLen: 1,
		},
		{
			name: "tag not found",
			tag:  "device",
			tv: []warnly.FieldValueNum{
				{Tag: "browser", Value: "Chrome", PercentsOfTotal: 50},
				{Tag: "os", Value: "Windows", PercentsOfTotal: 100},
			},
			wantLen: 0,
		},
		{
			name:    "empty slice",
			tag:     "browser",
			tv:      []warnly.FieldValueNum{},
			wantLen: 0,
		},
		{
			name:    "nil slice",
			tag:     "browser",
			tv:      nil,
			wantLen: 0,
		},
		{
			name: "all tags match",
			tag:  "env",
			tv: []warnly.FieldValueNum{
				{Tag: "env", Value: "prod", PercentsOfTotal: 50},
				{Tag: "env", Value: "staging", PercentsOfTotal: 30},
				{Tag: "env", Value: "dev", PercentsOfTotal: 20},
			},
			wantLen: 3,
		},
		{
			name: "case sensitive tag matching",
			tag:  "Browser",
			tv: []warnly.FieldValueNum{
				{Tag: "browser", Value: "Chrome", PercentsOfTotal: 50},
				{Tag: "Browser", Value: "Firefox", PercentsOfTotal: 50},
			},
			wantLen: 1,
		},
		{
			name: "tag at beginning",
			tag:  "first",
			tv: []warnly.FieldValueNum{
				{Tag: "first", Value: "value1", PercentsOfTotal: 40},
				{Tag: "second", Value: "value2", PercentsOfTotal: 30},
				{Tag: "third", Value: "value3", PercentsOfTotal: 30},
			},
			wantLen: 1,
		},
		{
			name: "tag at end",
			tag:  "last",
			tv: []warnly.FieldValueNum{
				{Tag: "first", Value: "value1", PercentsOfTotal: 40},
				{Tag: "second", Value: "value2", PercentsOfTotal: 30},
				{Tag: "last", Value: "value3", PercentsOfTotal: 30},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.ListTagValues(tt.tag, tt.tv)
			if len(result) != tt.wantLen {
				t.Errorf("ListTagValues(%q, ...) len = %d, want %d", tt.tag, len(result), tt.wantLen)
				return
			}
			for i, fvn := range result {
				if fvn.Tag != tt.tag {
					t.Errorf("ListTagValues(%q, ...)[%d].Tag = %q, want %q", tt.tag, i, fvn.Tag, tt.tag)
				}
			}
		})
	}
}

func TestIssueDetailsListTagValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		tag     string
		wantLen int
	}{
		{
			name: "tag with multiple values",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Tag: "browser", Value: "Chrome", PercentsOfTotal: 50},
					{Tag: "browser", Value: "Firefox", PercentsOfTotal: 30},
					{Tag: "browser", Value: "Safari", PercentsOfTotal: 20},
					{Tag: "os", Value: "Windows", PercentsOfTotal: 60},
					{Tag: "os", Value: "macOS", PercentsOfTotal: 40},
				},
			},
			tag:     "browser",
			wantLen: 3,
		},
		{
			name: "tag with single value",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Tag: "version", Value: "1.0.0", PercentsOfTotal: 100},
					{Tag: "environment", Value: "production", PercentsOfTotal: 100},
				},
			},
			tag:     "version",
			wantLen: 1,
		},
		{
			name: "tag not found",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Tag: "browser", Value: "Chrome", PercentsOfTotal: 50},
					{Tag: "os", Value: "Windows", PercentsOfTotal: 100},
				},
			},
			tag:     "device",
			wantLen: 0,
		},
		{
			name: "empty TagValueNum",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{},
			},
			tag:     "browser",
			wantLen: 0,
		},
		{
			name: "nil TagValueNum",
			details: &warnly.IssueDetails{
				TagValueNum: nil,
			},
			tag:     "browser",
			wantLen: 0,
		},
		{
			name: "all tags match",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Tag: "env", Value: "prod", PercentsOfTotal: 50},
					{Tag: "env", Value: "staging", PercentsOfTotal: 30},
					{Tag: "env", Value: "dev", PercentsOfTotal: 20},
				},
			},
			tag:     "env",
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.ListTagValues(tt.tag)
			if len(result) != tt.wantLen {
				t.Errorf("ListTagValues(%q) len = %d, want %d", tt.tag, len(result), tt.wantLen)
				return
			}
			for i, fvn := range result {
				if fvn.Tag != tt.tag {
					t.Errorf("ListTagValues(%q)[%d].Tag = %q, want %q", tt.tag, i, fvn.Tag, tt.tag)
				}
			}
		})
	}
}

func TestGetStackDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		event     *warnly.IssueEvent
		name      string
		wantLen   int
		wantEmpty bool
	}{
		{
			name: "single frame",
			event: &warnly.IssueEvent{
				ExceptionFramesAbsPath:  []string{"/app/main.go"},
				ExceptionFramesFunction: []string{"main"},
				ExceptionFramesLineno:   []int{42},
				ExceptionFramesInApp:    []int{1},
			},
			wantLen:   1,
			wantEmpty: false,
		},
		{
			name: "multiple frames",
			event: &warnly.IssueEvent{
				ExceptionFramesAbsPath:  []string{"/app/main.go", "/app/handler.go", "/lib/util.go"},
				ExceptionFramesFunction: []string{"main", "handleRequest", "process"},
				ExceptionFramesLineno:   []int{42, 100, 25},
				ExceptionFramesInApp:    []int{1, 1, 0},
			},
			wantLen:   3,
			wantEmpty: false,
		},
		{
			name: "empty exception frames",
			event: &warnly.IssueEvent{
				ExceptionFramesAbsPath:  []string{},
				ExceptionFramesFunction: []string{},
				ExceptionFramesLineno:   []int{},
				ExceptionFramesInApp:    []int{},
			},
			wantLen:   0,
			wantEmpty: true,
		},
		{
			name: "nil exception frames",
			event: &warnly.IssueEvent{
				ExceptionFramesAbsPath:  nil,
				ExceptionFramesFunction: nil,
				ExceptionFramesLineno:   nil,
				ExceptionFramesInApp:    nil,
			},
			wantLen:   0,
			wantEmpty: true,
		},
		{
			name: "frames with mixed InApp values",
			event: &warnly.IssueEvent{
				ExceptionFramesAbsPath:  []string{"/app/main.go", "/vendor/lib.go", "/app/handler.go"},
				ExceptionFramesFunction: []string{"main", "vendorFunc", "handleRequest"},
				ExceptionFramesLineno:   []int{42, 100, 25},
				ExceptionFramesInApp:    []int{1, 0, 1},
			},
			wantLen:   3,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.GetStackDetails(tt.event)

			if tt.wantEmpty {
				if result == nil {
					t.Errorf("GetStackDetails() returned nil, want empty slice")
				}
				if len(result) != 0 {
					t.Errorf("GetStackDetails() len = %d, want 0", len(result))
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("GetStackDetails() len = %d, want %d", len(result), tt.wantLen)
				return
			}

			if len(tt.event.ExceptionFramesAbsPath) > 0 {
				for i, detail := range result {
					originalIdx := tt.wantLen - 1 - i
					if detail.Filepath != tt.event.ExceptionFramesAbsPath[originalIdx] {
						t.Errorf("GetStackDetails()[%d].Filepath = %s, want %s", i, detail.Filepath, tt.event.ExceptionFramesAbsPath[originalIdx])
					}
					if detail.FunctionName != tt.event.ExceptionFramesFunction[originalIdx] {
						t.Errorf("GetStackDetails()[%d].FunctionName = %s, want %s", i, detail.FunctionName, tt.event.ExceptionFramesFunction[originalIdx])
					}
					if detail.LineNo != tt.event.ExceptionFramesLineno[originalIdx] {
						t.Errorf("GetStackDetails()[%d].LineNo = %d, want %d", i, detail.LineNo, tt.event.ExceptionFramesLineno[originalIdx])
					}
					expectedInApp := tt.event.ExceptionFramesInApp[originalIdx] == 1
					if detail.InApp != expectedInApp {
						t.Errorf("GetStackDetails()[%d].InApp = %v, want %v", i, detail.InApp, expectedInApp)
					}
				}
			}
		})
	}
}

func TestIssueDetailsStackHidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		details *warnly.IssueDetails
		name    string
		want    int
		wantNil bool
	}{
		{
			name: "more than 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
					{Filepath: "/lib/util.go", LineNo: 4},
					{Filepath: "/lib/helper.go", LineNo: 5},
					{Filepath: "/vendor/pkg.go", LineNo: 6},
					{Filepath: "/vendor/lib.go", LineNo: 7},
					{Filepath: "/vendor/other.go", LineNo: 8},
				},
			},
			want:    3,
			wantNil: false,
		},
		{
			name: "exactly 6 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
					{Filepath: "/lib/util.go", LineNo: 4},
					{Filepath: "/lib/helper.go", LineNo: 5},
					{Filepath: "/vendor/pkg.go", LineNo: 6},
				},
			},
			want:    1,
			wantNil: false,
		},
		{
			name: "exactly 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
					{Filepath: "/lib/util.go", LineNo: 4},
					{Filepath: "/lib/helper.go", LineNo: 5},
				},
			},
			want:    0,
			wantNil: true,
		},
		{
			name: "less than 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
				},
			},
			want:    0,
			wantNil: true,
		},
		{
			name: "single stack detail",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
				},
			},
			want:    0,
			wantNil: true,
		},
		{
			name: "empty stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{},
			},
			want:    0,
			wantNil: true,
		},
		{
			name: "nil stack details",
			details: &warnly.IssueDetails{
				StackDetails: nil,
			},
			want:    0,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.StackHidden()
			if tt.wantNil {
				if result != nil {
					t.Errorf("StackHidden() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("StackHidden() = nil, want non-nil")
					return
				}
				if len(result) != tt.want {
					t.Errorf("StackHidden() len = %d, want %d", len(result), tt.want)
					return
				}
				if len(tt.details.StackDetails) > 5 {
					for i := range result {
						if result[i].Filepath != tt.details.StackDetails[i+5].Filepath {
							t.Errorf("StackHidden()[%d].Filepath = %s, want %s", i, result[i].Filepath, tt.details.StackDetails[i+5].Filepath)
						}
					}
				}
			}
		})
	}
}

func TestIssueDetailsStackVisible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		details *warnly.IssueDetails
		name    string
		want    int
	}{
		{
			name: "more than 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
					{Filepath: "/lib/util.go", LineNo: 4},
					{Filepath: "/lib/helper.go", LineNo: 5},
					{Filepath: "/vendor/pkg.go", LineNo: 6},
					{Filepath: "/vendor/lib.go", LineNo: 7},
				},
			},
			want: 5,
		},
		{
			name: "exactly 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
					{Filepath: "/lib/util.go", LineNo: 4},
					{Filepath: "/lib/helper.go", LineNo: 5},
				},
			},
			want: 5,
		},
		{
			name: "less than 5 stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
					{Filepath: "/app/handler.go", LineNo: 2},
					{Filepath: "/app/service.go", LineNo: 3},
				},
			},
			want: 3,
		},
		{
			name: "single stack detail",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 1},
				},
			},
			want: 1,
		},
		{
			name: "empty stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{},
			},
			want: 0,
		},
		{
			name: "nil stack details",
			details: &warnly.IssueDetails{
				StackDetails: nil,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.StackVisible()
			if len(result) != tt.want {
				t.Errorf("StackVisible() len = %d, want %d", len(result), tt.want)
				return
			}
			if len(tt.details.StackDetails) > 5 {
				for i := range 5 {
					if result[i].Filepath != tt.details.StackDetails[i].Filepath {
						t.Errorf("StackVisible()[%d].Filepath = %s, want %s", i, result[i].Filepath, tt.details.StackDetails[i].Filepath)
					}
				}
			}
		})
	}
}

func TestIssueDetailsHasStackDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		details *warnly.IssueDetails
		name    string
		want    bool
	}{
		{
			name: "has single stack detail",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{
						Filepath:     "/app/main.go",
						FunctionName: "main",
						LineNo:       42,
						InApp:        true,
					},
				},
			},
			want: true,
		},
		{
			name: "has multiple stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{
					{Filepath: "/app/main.go", LineNo: 42},
					{Filepath: "/app/handler.go", LineNo: 100},
					{Filepath: "/lib/util.go", LineNo: 25},
				},
			},
			want: true,
		},
		{
			name: "empty stack details",
			details: &warnly.IssueDetails{
				StackDetails: []warnly.StackDetail{},
			},
			want: false,
		},
		{
			name: "nil stack details",
			details: &warnly.IssueDetails{
				StackDetails: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.HasStackDetails()
			if result != tt.want {
				t.Errorf("HasStackDetails() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestStackDetailInAppStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		detail *warnly.StackDetail
		want   string
	}{
		{
			name: "InApp is true",
			detail: &warnly.StackDetail{
				Filepath:     "/app/main.go",
				FunctionName: "main",
				LineNo:       42,
				InApp:        true,
			},
			want: "In App",
		},
		{
			name: "InApp is false",
			detail: &warnly.StackDetail{
				Filepath:     "/vendor/lib.go",
				FunctionName: "someFunc",
				LineNo:       100,
				InApp:        false,
			},
			want: "",
		},
		{
			name: "InApp false with minimal detail",
			detail: &warnly.StackDetail{
				InApp: false,
			},
			want: "",
		},
		{
			name: "InApp true with minimal detail",
			detail: &warnly.StackDetail{
				InApp: true,
			},
			want: "In App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.detail.InAppStr()
			if result != tt.want {
				t.Errorf("InAppStr() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestIssueDetailsEventID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		want    string
	}{
		{
			name: "valid event id",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					EventID: "550e8400-e29b-41d4-a716-446655440000",
				},
			},
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "empty event id",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					EventID: "",
				},
			},
			want: "",
		},
		{
			name: "uuid format event id",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					EventID: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
				},
			},
			want: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.EventID()
			if result != tt.want {
				t.Errorf("EventID() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestIssueDetailsProgressLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		val     string
		want    string
	}{
		{
			name: "percent >= 100",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "Chrome", PercentsOfTotal: 100},
					{Value: "Firefox", PercentsOfTotal: 50},
				},
			},
			val:  "Chrome",
			want: "w-full",
		},
		{
			name: "percent >= 75",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "iOS", PercentsOfTotal: 80},
					{Value: "Android", PercentsOfTotal: 20},
				},
			},
			val:  "iOS",
			want: "w-3/4",
		},
		{
			name: "percent >= 50",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "Linux", PercentsOfTotal: 60},
					{Value: "Windows", PercentsOfTotal: 40},
				},
			},
			val:  "Linux",
			want: "w-1/2",
		},
		{
			name: "percent >= 25",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "Edge", PercentsOfTotal: 30},
					{Value: "Safari", PercentsOfTotal: 70},
				},
			},
			val:  "Edge",
			want: "w-1/4",
		},
		{
			name: "percent < 25",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "Opera", PercentsOfTotal: 10},
					{Value: "Chrome", PercentsOfTotal: 90},
				},
			},
			val:  "Opera",
			want: "w-1/5",
		},
		{
			name: "value not found",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "Chrome", PercentsOfTotal: 50},
					{Value: "Firefox", PercentsOfTotal: 50},
				},
			},
			val:  "Safari",
			want: "",
		},
		{
			name: "empty TagValueNum",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{},
			},
			val:  "Chrome",
			want: "",
		},
		{
			name: "exact boundary 75",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "test", PercentsOfTotal: 75},
				},
			},
			val:  "test",
			want: "w-3/4",
		},
		{
			name: "exact boundary 50",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "test", PercentsOfTotal: 50},
				},
			},
			val:  "test",
			want: "w-1/2",
		},
		{
			name: "exact boundary 25",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "test", PercentsOfTotal: 25},
				},
			},
			val:  "test",
			want: "w-1/4",
		},
		{
			name: "zero percent",
			details: &warnly.IssueDetails{
				TagValueNum: []warnly.FieldValueNum{
					{Value: "test", PercentsOfTotal: 0},
				},
			},
			val:  "test",
			want: "w-1/5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.ProgressLen(tt.val)
			if result != tt.want {
				t.Errorf("ProgressLen(%q) = %q, want %q", tt.val, result, tt.want)
			}
		})
	}
}

func TestIssueDetailsTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		tag     string
		want    string
	}{
		{
			name: "tag found",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "os", "version"},
					TagsValue: []string{"Chrome", "Windows", "1.0"},
				},
			},
			tag:  "os",
			want: "Windows",
		},
		{
			name: "tag not found",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "os"},
					TagsValue: []string{"Chrome", "Windows"},
				},
			},
			tag:  "device",
			want: "",
		},
		{
			name: "empty tags",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{},
					TagsValue: []string{},
				},
			},
			tag:  "browser",
			want: "",
		},
		{
			name: "first tag matches",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "os", "version"},
					TagsValue: []string{"Firefox", "Linux", "2.0"},
				},
			},
			tag:  "browser",
			want: "Firefox",
		},
		{
			name: "last tag matches",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "os", "version"},
					TagsValue: []string{"Safari", "macOS", "3.0"},
				},
			},
			tag:  "version",
			want: "3.0",
		},
		{
			name: "tag with empty value",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "device"},
					TagsValue: []string{"Chrome", ""},
				},
			},
			tag:  "device",
			want: "",
		},
		{
			name: "single tag found",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"env"},
					TagsValue: []string{"production"},
				},
			},
			tag:  "env",
			want: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.Tag(tt.tag)
			if result != tt.want {
				t.Errorf("Tag(%q) = %q, want %q", tt.tag, result, tt.want)
			}
		})
	}
}

func TestIssueDetailsContexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		details *warnly.IssueDetails
		want    map[string]string
		name    string
	}{
		{
			name: "empty contexts",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					ContextsKey:   []string{},
					ContextsValue: []string{},
				},
			},
			want: nil,
		},
		{
			name: "single context",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					ContextsKey:   []string{"device"},
					ContextsValue: []string{"iPhone 14"},
				},
			},
			want: map[string]string{
				"device": "iPhone 14",
			},
		},
		{
			name: "multiple contexts",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					ContextsKey:   []string{"os", "device", "app_version"},
					ContextsValue: []string{"iOS 17.0", "iPad Pro", "2.5.1"},
				},
			},
			want: map[string]string{
				"os":          "iOS 17.0",
				"device":      "iPad Pro",
				"app_version": "2.5.1",
			},
		},
		{
			name: "contexts with empty values",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					ContextsKey:   []string{"key1", "key2"},
					ContextsValue: []string{"", "value2"},
				},
			},
			want: map[string]string{
				"key1": "",
				"key2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.Contexts()
			if len(result) != len(tt.want) {
				t.Errorf("Contexts() len = %d, want %d", len(result), len(tt.want))
				return
			}
			for key, wantVal := range tt.want {
				gotVal, ok := result[key]
				if !ok {
					t.Errorf("Contexts() missing key %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("Contexts()[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestIssueDetailsTagKeyValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		want    []warnly.TagKeyValue
	}{
		{
			name: "empty tags",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{},
					TagsValue: []string{},
				},
			},
			want: nil,
		},
		{
			name: "single tag",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser"},
					TagsValue: []string{"Chrome"},
				},
			},
			want: []warnly.TagKeyValue{
				{Key: "browser", Value: "Chrome"},
			},
		},
		{
			name: "multiple tags",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"browser", "os", "version"},
					TagsValue: []string{"Firefox", "macOS", "1.0.0"},
				},
			},
			want: []warnly.TagKeyValue{
				{Key: "browser", Value: "Firefox"},
				{Key: "os", Value: "macOS"},
				{Key: "version", Value: "1.0.0"},
			},
		},
		{
			name: "tags with empty values",
			details: &warnly.IssueDetails{
				LastEvent: &warnly.IssueEvent{
					TagsKey:   []string{"key1", "key2"},
					TagsValue: []string{"", "value2"},
				},
			},
			want: []warnly.TagKeyValue{
				{Key: "key1", Value: ""},
				{Key: "key2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.TagKeyValue()
			if len(result) != len(tt.want) {
				t.Errorf("TagKeyValue() len = %d, want %d", len(result), len(tt.want))
				return
			}
			for i, got := range result {
				if got != tt.want[i] {
					t.Errorf("TagKeyValue()[%d] = %+v, want %+v", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestCut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want string
		n    int
	}{
		{
			name: "string shorter than limit",
			s:    "hello",
			n:    10,
			want: "hello",
		},
		{
			name: "string equal to limit",
			s:    "hello",
			n:    5,
			want: "hello",
		},
		{
			name: "string longer than limit",
			s:    "hello world",
			n:    5,
			want: "hello...",
		},
		{
			name: "empty string",
			s:    "",
			n:    10,
			want: "",
		},
		{
			name: "limit is 0",
			s:    "hello",
			n:    0,
			want: "...",
		},
		{
			name: "limit is 1",
			s:    "hello",
			n:    1,
			want: "h...",
		},
		{
			name: "very long string",
			s:    "The quick brown fox jumps over the lazy dog",
			n:    10,
			want: "The quick ...",
		},
		{
			name: "unicode characters",
			s:    "hello привет",
			n:    5,
			want: "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.Cut(tt.s, tt.n)
			if result != tt.want {
				t.Errorf("Cut(%q, %d) = %q, want %q", tt.s, tt.n, result, tt.want)
			}
		})
	}
}

func TestIssueDetailsGetPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details *warnly.IssueDetails
		want    string
	}{
		{
			name: "Go platform",
			details: &warnly.IssueDetails{
				Platform: warnly.PlatformGolang,
			},
			want: "Go",
		},
		{
			name: "unknown platform",
			details: &warnly.IssueDetails{
				Platform: 0,
			},
			want: "unknown",
		},
		{
			name: "invalid platform",
			details: &warnly.IssueDetails{
				Platform: 99,
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.details.GetPlatform()
			if result != tt.want {
				t.Errorf("GetPlatform() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"", 0, false},
		{"2h", 2 * time.Hour, false},
		{"5m", 5 * time.Minute, false},
		{"10s", 10 * time.Second, false},
		{"3d", 3 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"100h", 100 * time.Hour, false},
		{"0h", 0, false},
		{"1x", 0, true},
		{"abc", 0, true},
		{"5", 0, true},
		{"-2h", 2 * time.Hour, false},
	}

	for _, tt := range tests {
		got, err := warnly.ParseDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Fatalf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseTimeRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantStart time.Time
		wantEnd   time.Time
		start     string
		end       string
		wantErr   bool
	}{
		{
			start:     "2025-06-20T00:00:00",
			end:       "2025-06-26T23:59:59",
			wantStart: time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2025, 6, 26, 23, 59, 59, 0, time.UTC),
			wantErr:   false,
		},
		{
			start:     "2024-01-01T12:00:00",
			end:       "2024-01-02T12:00:00",
			wantStart: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			start:   "invalid",
			end:     "2025-06-26T23:59:59",
			wantErr: true,
		},
		{
			start:   "2025-06-20T00:00:00",
			end:     "invalid",
			wantErr: true,
		},
		{
			start:   "",
			end:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		gotStart, gotEnd, err := warnly.ParseTimeRange(tt.start, tt.end)
		if tt.wantErr {
			require.Error(t, err, "ParseTimeRange(%q, %q) expected error", tt.start, tt.end)
			continue
		}
		require.NoError(t, err, "ParseTimeRange(%q, %q) unexpected error", tt.start, tt.end)
		require.True(t, gotStart.Equal(tt.wantStart), "ParseTimeRange(%q, %q) gotStart = %v, want %v", tt.start, tt.end, gotStart, tt.wantStart)
		require.True(t, gotEnd.Equal(tt.wantEnd), "ParseTimeRange(%q, %q) gotEnd = %v, want %v", tt.start, tt.end, gotEnd, tt.wantEnd)
	}
}

func TestParseQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected []warnly.QueryToken
	}{
		{
			name:     "empty query",
			query:    "",
			expected: []warnly.QueryToken{},
		},
		{
			name:  "single tag",
			query: "release:nordland@0.1.0",
			expected: []warnly.QueryToken{
				{Key: "release", Operator: "is", Value: "nordland@0.1.0"},
			},
		},
		{
			name:  "tag with not",
			query: "server_name:!Olegs-MacBook-Pro.local",
			expected: []warnly.QueryToken{
				{Key: "server_name", Operator: "is not", Value: "Olegs-MacBook-Pro.local"},
			},
		},
		{
			name:  "tag with quoted value",
			query: `server_name:"delein computer"`,
			expected: []warnly.QueryToken{
				{Key: "server_name", Operator: "is", Value: "delein computer"},
			},
		},
		{
			name:  "raw text",
			query: `"pro error"`,
			expected: []warnly.QueryToken{
				{Value: "pro error", IsRawText: true},
			},
		},
		{
			name:  "multiple tokens",
			query: `release:nordland@0.1.0 level:error server_name:!Olegs-MacBook-Pro.local "pro error" "div error" server_name:"delein computer"`,
			expected: []warnly.QueryToken{
				{Key: "release", Operator: "is", Value: "nordland@0.1.0"},
				{Key: "level", Operator: "is", Value: "error"},
				{Key: "server_name", Operator: "is not", Value: "Olegs-MacBook-Pro.local"},
				{Value: "pro error", IsRawText: true},
				{Value: "div error", IsRawText: true},
				{Key: "server_name", Operator: "is", Value: "delein computer"},
			},
		},
		{
			name:  "single quotes",
			query: "tag:'value with spaces'",
			expected: []warnly.QueryToken{
				{Key: "tag", Operator: "is", Value: "value with spaces"},
			},
		},
		{
			name:  "mixed quotes",
			query: `"text" key:value`,
			expected: []warnly.QueryToken{
				{Value: "text", IsRawText: true},
				{Key: "key", Operator: "is", Value: "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.ParseQuery(tt.query)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tokens, got %d", len(tt.expected), len(result))
				return
			}
			for i, token := range result {
				expected := tt.expected[i]
				if token != expected {
					t.Errorf("token %d: expected %+v, got %+v", i, expected, token)
				}
			}
		})
	}
}

func TestTeammateAvatarInitials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		teammate *warnly.Teammate
		want     string
	}{
		{
			name: "normal case",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "Doe",
			},
			want: "JD",
		},
		{
			name: "single character names",
			teammate: &warnly.Teammate{
				Name:    "A",
				Surname: "B",
			},
			want: "AB",
		},
		{
			name: "long names",
			teammate: &warnly.Teammate{
				Name:    "Alexander",
				Surname: "Sokolov",
			},
			want: "AS",
		},
		{
			name: "lowercase names",
			teammate: &warnly.Teammate{
				Name:    "john",
				Surname: "doe",
			},
			want: "JD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.teammate.AvatarInitials()
			if result != tt.want {
				t.Errorf("AvatarInitials() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestListIssuesResultNoIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result   *warnly.ListIssuesResult
		name     string
		wantTrue bool
	}{
		{
			name:     "empty issues",
			result:   &warnly.ListIssuesResult{Issues: []warnly.IssueEntry{}},
			wantTrue: true,
		},
		{
			name:     "nil issues",
			result:   &warnly.ListIssuesResult{Issues: nil},
			wantTrue: true,
		},
		{
			name: "single issue",
			result: &warnly.ListIssuesResult{
				Issues: []warnly.IssueEntry{
					{ID: 1, Message: "Error 1"},
				},
			},
			wantTrue: false,
		},
		{
			name: "multiple issues",
			result: &warnly.ListIssuesResult{
				Issues: []warnly.IssueEntry{
					{ID: 1, Message: "Error 1"},
					{ID: 2, Message: "Error 2"},
					{ID: 3, Message: "Error 3"},
				},
			},
			wantTrue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.result.NoIssues()
			if result != tt.wantTrue {
				t.Errorf("NoIssues() = %v, want %v", result, tt.wantTrue)
			}
		})
	}
}

func TestTeammateFullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		teammate *warnly.Teammate
		want     string
	}{
		{
			name: "normal case",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "Doe",
			},
			want: "John Doe",
		},
		{
			name: "single character names",
			teammate: &warnly.Teammate{
				Name:    "A",
				Surname: "B",
			},
			want: "A B",
		},
		{
			name: "long names",
			teammate: &warnly.Teammate{
				Name:    "Alexander",
				Surname: "Sokolov",
			},
			want: "Alexander Sokolov",
		},
		{
			name: "mixed case",
			teammate: &warnly.Teammate{
				Name:    "john",
				Surname: "DOE",
			},
			want: "john DOE",
		},
		{
			name: "empty name",
			teammate: &warnly.Teammate{
				Name:    "",
				Surname: "Doe",
			},
			want: " Doe",
		},
		{
			name: "empty surname",
			teammate: &warnly.Teammate{
				Name:    "John",
				Surname: "",
			},
			want: "John ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.teammate.FullName()
			if result != tt.want {
				t.Errorf("FullName() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	mockNow := func() time.Time { return now }

	tests := []struct {
		t          time.Time
		name       string
		wantWide   string
		wantNarrow string
		narrow     bool
	}{
		{
			name:       "30 seconds ago",
			t:          now.Add(-30 * time.Second),
			narrow:     false,
			wantWide:   "30 seconds",
			wantNarrow: "30sec",
		},
		{
			name:       "1 second ago",
			t:          now.Add(-time.Second),
			narrow:     false,
			wantWide:   "1 second",
			wantNarrow: "1sec",
		},
		{
			name:       "5 minutes ago",
			t:          now.Add(-5 * time.Minute),
			narrow:     false,
			wantWide:   "5 minutes",
			wantNarrow: "5min",
		},
		{
			name:       "2 hours ago",
			t:          now.Add(-2 * time.Hour),
			narrow:     false,
			wantWide:   "2 hours",
			wantNarrow: "2h",
		},
		{
			name:       "10 days ago",
			t:          now.Add(-10 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "10 days",
			wantNarrow: "10d",
		},
		{
			name:       "2 months ago",
			t:          now.Add(-60 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "2 months",
			wantNarrow: "2mo",
		},
		{
			name:       "2 years ago",
			t:          now.Add(-730 * 24 * time.Hour),
			narrow:     false,
			wantWide:   "2 years",
			wantNarrow: "2y",
		},
		{
			name:       "30 seconds ago (narrow)",
			t:          now.Add(-30 * time.Second),
			narrow:     true,
			wantWide:   "",
			wantNarrow: "30sec",
		},
		{
			name:       "5 minutes ago (narrow)",
			t:          now.Add(-5 * time.Minute),
			narrow:     true,
			wantWide:   "",
			wantNarrow: "5min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := warnly.TimeAgo(mockNow, tt.t, tt.narrow)
			want := tt.wantWide
			if tt.narrow {
				want = tt.wantNarrow
			}
			if result != want {
				t.Errorf("TimeAgo() = %q, want %q", result, want)
			}
		})
	}
}
