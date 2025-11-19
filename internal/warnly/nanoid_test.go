package warnly_test

import (
	"strings"
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

func TestValidateNanoID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		id        string
		errMsg    string
		wantErr   bool
	}{
		{
			name:      "valid id",
			fieldName: "testID",
			id:        "abc1234",
			wantErr:   false,
		},
		{
			name:      "empty id",
			fieldName: "testID",
			id:        "",
			wantErr:   true,
			errMsg:    "testID cannot be blank",
		},
		{
			name:      "too short",
			fieldName: "testID",
			id:        "abc",
			wantErr:   true,
			errMsg:    "testID should be 7 characters long",
		},
		{
			name:      "too long",
			fieldName: "testID",
			id:        "abc12345",
			wantErr:   true,
			errMsg:    "testID should be 7 characters long",
		},
		{
			name:      "invalid characters",
			fieldName: "testID",
			id:        "abc!234",
			wantErr:   true,
			errMsg:    "testID has invalid characters",
		},
		{
			name:      "uppercase invalid",
			fieldName: "testID",
			id:        "ABC1234",
			wantErr:   true,
			errMsg:    "testID has invalid characters",
		},
		{
			name:      "valid with all numbers",
			fieldName: "testID",
			id:        "0123456",
			wantErr:   false,
		},
		{
			name:      "valid with all letters",
			fieldName: "testID",
			id:        "abcdefg",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := warnly.ValidateNanoID(tt.fieldName, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateNanoID() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateNanoID() error = %v, want containing %v", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("ValidateNanoID() unexpected error = %v", err)
			}
		})
	}
}

func TestNewNanoID(t *testing.T) {
	t.Parallel()

	id1, err := warnly.NewNanoID()
	if err != nil {
		t.Errorf("NewNanoID() error = %v, wantErr false", err)
		return
	}

	id2, err := warnly.NewNanoID()
	if err != nil {
		t.Errorf("NewNanoID() error = %v, wantErr false", err)
		return
	}

	if len(id1) != 7 {
		t.Errorf("NewNanoID() len = %d, want 7", len(id1))
	}
	if len(id2) != 7 {
		t.Errorf("NewNanoID() len = %d, want 7", len(id2))
	}
	if id1 == id2 {
		t.Errorf("NewNanoID() generated same ID twice: %v", id1)
	}

	if err := warnly.ValidateNanoID("id", id1); err != nil {
		t.Errorf("NewNanoID() generated invalid ID: %v", err)
	}
	if err := warnly.ValidateNanoID("id", id2); err != nil {
		t.Errorf("NewNanoID() generated invalid ID: %v", err)
	}
}

func TestMustNanoID(t *testing.T) {
	t.Parallel()

	id1 := warnly.MustNanoID()
	id2 := warnly.MustNanoID()

	if len(id1) != 7 {
		t.Errorf("MustNanoID() len = %d, want 7", len(id1))
	}
	if len(id2) != 7 {
		t.Errorf("MustNanoID() len = %d, want 7", len(id2))
	}
	if id1 == id2 {
		t.Errorf("MustNanoID() generated same ID twice: %v", id1)
	}

	if err := warnly.ValidateNanoID("id", id1); err != nil {
		t.Errorf("MustNanoID() generated invalid ID: %v", err)
	}
	if err := warnly.ValidateNanoID("id", id2); err != nil {
		t.Errorf("MustNanoID() generated invalid ID: %v", err)
	}
}
