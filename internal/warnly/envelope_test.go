package warnly_test

import (
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

func TestGetExceptionStackTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []string
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []string{},
		},
		{
			name: fixtureSingleException,
			exceptions: []warnly.Exception{
				{Type: fixtureTypeError},
			},
			want: []string{fixtureTypeError},
		},
		{
			name: fixtureMultipleExceptions,
			exceptions: []warnly.Exception{
				{Type: fixtureTypeError},
				{Type: "SyntaxError"},
			},
			want: []string{fixtureTypeError, "SyntaxError"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := warnly.GetExceptionStackTypes(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionStackTypes() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionStackTypes()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetExceptionStackValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []string
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []string{},
		},
		{
			name: fixtureSingleException,
			exceptions: []warnly.Exception{
				{Value: "some error"},
			},
			want: []string{"some error"},
		},
		{
			name: fixtureMultipleExceptions,
			exceptions: []warnly.Exception{
				{Value: "error1"},
				{Value: "error2"},
			},
			want: []string{"error1", "error2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := warnly.GetExceptionStackValues(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionStackValues() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionStackValues()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetExceptionFramesAbsPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []string
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []string{},
		},
		{
			name: fixtureSingleExceptionWithFrames,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{AbsPath: fixtureExceptionFile},
							{AbsPath: "/path/to/file2.go"},
						},
					},
				},
			},
			want: []string{fixtureExceptionFile, "/path/to/file2.go"},
		},
		{
			name: fixtureMultipleExceptions,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{AbsPath: "/path1"},
						},
					},
				},
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{AbsPath: "/path2"},
							{AbsPath: "/path3"},
						},
					},
				},
			},
			want: []string{"/path1", "/path2", "/path3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionFramesAbsPath(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionFramesAbsPath() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionFramesAbsPath()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetBreaker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		want       string
		exceptions []warnly.Exception
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       "",
		},
		{
			name: "exception with no stack trace",
			exceptions: []warnly.Exception{
				{Type: "Error", Value: fixtureMsg},
			},
			want: "",
		},
		{
			name: "single frame not ignored",
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{Module: "mymodule", Function: "myfunc"},
						},
					},
				},
			},
			want: "mymodule in myfunc",
		},
		{
			name: "multiple frames, last not ignored",
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{Module: "github.com/buger/jsonparser", Function: "parse"},
							{Module: "mymodule", Function: "myfunc"},
						},
					},
				},
			},
			want: "mymodule in myfunc",
		},
		{
			name: "all frames ignored",
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{Module: "github.com/buger/jsonparser", Function: "parse"},
							{Module: "github.com/rs/zerolog", Function: "log"},
						},
					},
				},
			},
			want: "github.com/buger/jsonparser in parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetBreaker(tt.exceptions)
			if got != tt.want {
				t.Errorf("GetBreaker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExceptionValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		defaultVal string
		want       string
		exceptions []warnly.Exception
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			defaultVal: fixtureDefault,
			want:       fixtureDefault,
		},
		{
			name: fixtureSingleException,
			exceptions: []warnly.Exception{
				{Value: "error msg"},
			},
			defaultVal: fixtureDefault,
			want:       "error msg",
		},
		{
			name: fixtureMultipleExceptions,
			exceptions: []warnly.Exception{
				{Value: fixtureFirst},
				{Value: fixtureLast},
			},
			defaultVal: fixtureDefault,
			want:       fixtureLast,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionValue(tt.exceptions, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetExceptionValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExceptionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		defaultVal string
		want       string
		exceptions []warnly.Exception
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			defaultVal: fixtureDefault,
			want:       fixtureDefault,
		},
		{
			name: "exception with type",
			exceptions: []warnly.Exception{
				{Type: fixtureTypeError, Value: fixtureMsg},
			},
			defaultVal: fixtureDefault,
			want:       fixtureTypeError,
		},
		{
			name: "exception with value but no type",
			exceptions: []warnly.Exception{
				{Type: "", Value: fixtureMsg},
			},
			defaultVal: fixtureDefault,
			want:       "Error",
		},
		{
			name: "exception with no type and no value",
			exceptions: []warnly.Exception{
				{Type: "", Value: ""},
			},
			defaultVal: fixtureDefault,
			want:       fixtureDefault,
		},
		{
			name: "multiple exceptions, last has type",
			exceptions: []warnly.Exception{
				{Type: "First", Value: fixtureFirst},
				{Type: "Last", Value: fixtureLast},
			},
			defaultVal: fixtureDefault,
			want:       "Last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionType(tt.exceptions, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetExceptionType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExceptionFramesColNo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []uint32
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []uint32{},
		},
		{
			name: fixtureSingleExceptionWithFrames,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{LineNo: 10},
							{LineNo: 20},
						},
					},
				},
			},
			want: []uint32{10, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionFramesColNo(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionFramesColNo() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionFramesColNo()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetExceptionFramesFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []string
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []string{},
		},
		{
			name: fixtureSingleExceptionWithFrames,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{AbsPath: fixtureExceptionFile},
							{AbsPath: "file2.go"},
						},
					},
				},
			},
			want: []string{"file1.go", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionFramesFilename(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionFramesFilename() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionFramesFilename()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetExceptionFramesFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []string
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []string{},
		},
		{
			name: fixtureSingleExceptionWithFrames,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{Function: "func1"},
							{Function: "func2"},
						},
					},
				},
			},
			want: []string{"func1", "func2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionFramesFunction(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionFramesFunction() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionFramesFunction()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetExceptionFramesLineNo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		exceptions []warnly.Exception
		want       []uint32
	}{
		{
			name:       fixtureEmptyExceptions,
			exceptions: []warnly.Exception{},
			want:       []uint32{},
		},
		{
			name: fixtureSingleExceptionWithFrames,
			exceptions: []warnly.Exception{
				{
					StackTrace: warnly.StackTrace{
						Frames: []warnly.Frame{
							{LineNo: 10},
							{LineNo: 20},
						},
					},
				},
			},
			want: []uint32{10, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetExceptionFramesLineNo(tt.exceptions)
			if len(got) != len(tt.want) {
				t.Errorf("GetExceptionFramesLineNo() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("GetExceptionFramesLineNo()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestGetLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
		want  warnly.Level
	}{
		{
			name:  "fatal",
			level: "fatal",
			want:  warnly.LevelFatal,
		},
		{
			name:  fixtureError,
			level: fixtureError,
			want:  warnly.LevelError,
		},
		{
			name:  "warning",
			level: "warning",
			want:  warnly.LevelWarning,
		},
		{
			name:  "info",
			level: "info",
			want:  warnly.LevelInfo,
		},
		{
			name:  "debug",
			level: "debug",
			want:  warnly.LevelDebug,
		},
		{
			name:  "trace",
			level: "trace",
			want:  warnly.LevelTrace,
		},
		{
			name:  fixtureUnknown,
			level: fixtureUnknown,
			want:  warnly.LevelUnknown,
		},
		{
			name:  fixtureInvalid,
			level: fixtureInvalid,
			want:  warnly.LevelUnknown,
		},
		{
			name:  "case insensitive",
			level: "ERROR",
			want:  warnly.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := warnly.GetLevel(tt.level)
			if got != tt.want {
				t.Errorf("GetLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
