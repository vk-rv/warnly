package warnly_test

import (
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

func TestParseUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
			wantLen: 16,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "invalid UUID",
			input:   "invalid-uuid",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "UUID without hyphens",
			input:   "550e8400e29b41d4a716446655440000",
			wantErr: false,
			wantLen: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := warnly.ParseUUID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("ParseUUID() len(got) = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestNewUUID(t *testing.T) {
	t.Parallel()

	u1 := warnly.NewUUID()
	u2 := warnly.NewUUID()

	if len(u1) != 16 {
		t.Errorf("NewUUID() len = %v, want 16", len(u1))
	}
	if len(u2) != 16 {
		t.Errorf("NewUUID() len = %v, want 16", len(u2))
	}
	if string(u1) == string(u2) {
		t.Errorf("NewUUID() generated same UUID twice: %v", u1)
	}
}

func TestUUID_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		uuid warnly.UUID
	}{
		{
			name: "valid UUID",
			uuid: warnly.UUID{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00},
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "nil UUID",
			uuid: nil,
			want: "",
		},
		{
			name: "short UUID",
			uuid: warnly.UUID{1, 2, 3},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.uuid.String(); got != tt.want {
				t.Errorf("UUID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUUID_Scan(t *testing.T) {
	t.Parallel()

	//nolint:govet // ignore
	tests := []struct {
		wantErr bool
		want    warnly.UUID
		name    string
		src     any
	}{
		{
			name:    "nil src",
			src:     nil,
			wantErr: false,
			want:    nil,
		},
		{
			name:    "valid []byte",
			src:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			wantErr: false,
			want:    warnly.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			name:    "unsupported type",
			src:     "string",
			wantErr: true,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var u warnly.UUID
			err := u.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("UUID.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(u) != string(tt.want) {
				t.Errorf("UUID.Scan() = %v, want %v", u, tt.want)
			}
		})
	}
}

func TestUUID_Value(t *testing.T) {
	t.Parallel()

	u := warnly.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	val, err := u.Value()
	if err != nil {
		t.Errorf("UUID.Value() error = %v", err)
		return
	}
	if v, ok := val.([]byte); !ok || string(v) != string(u) {
		t.Errorf("UUID.Value() = %v, want %v", v, u)
	}
}

func TestUUIDPtr(t *testing.T) {
	t.Parallel()

	u := warnly.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	ptr := warnly.UUIDPtr(u)

	if ptr == nil {
		t.Errorf("UUIDPtr() returned nil")
		return
	}
	if string(*ptr) != string(u) {
		t.Errorf("UUIDPtr() dereference = %v, want %v", *ptr, u)
	}
}
