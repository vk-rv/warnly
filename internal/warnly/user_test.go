package warnly_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestUser_AvatarInitials(t *testing.T) {
	t.Parallel()

	user := &warnly.User{
		Name:    "John",
		Surname: "Doe",
	}

	result := user.AvatarInitials()
	require.Equal(t, "JD", result)
}

func TestUser_FullName(t *testing.T) {
	t.Parallel()

	user := &warnly.User{
		Name:    "John",
		Surname: "Doe",
	}

	result := user.FullName()
	require.Equal(t, "John Doe", result)
}

func TestUsernameFromEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "john.doe@example.com",
			want:    "john.doe",
			wantErr: false,
		},
		{
			name:    "email without subdomain",
			email:   "test@gmail.com",
			want:    "test",
			wantErr: false,
		},
		{
			name:    "email with numbers",
			email:   "user123@domain.org",
			want:    "user123",
			wantErr: false,
		},
		{
			name:    "no @ symbol",
			email:   "invalidemail",
			want:    "",
			wantErr: true,
		},
		{
			name:    "@ at the beginning",
			email:   "@example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "@ at the end",
			email:   "user@",
			want:    "user",
			wantErr: false,
		},
		{
			name:    "multiple @",
			email:   "user@domain@com",
			want:    "user",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := warnly.UsernameFromEmail(tt.email)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, result)
			}
		})
	}
}
