package mysql_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestGetProject(t *testing.T) {
	t.Parallel()

	const query = `SELECT id, created_at, name, user_id, team_id, platform, project_key FROM project WHERE id = ?`

	date := time.Date(2025, 1, 29, 6, 47, 9, 0, time.UTC)

	queryError := errors.New("query error")

	tests := []struct {
		expectedError   error
		mockExpect      func(mock sqlmock.Sqlmock)
		expectedProject *warnly.Project
		name            string
	}{
		{
			name: "HappyPath",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "name", "user_id", "team_id", "platform", "project_key"}).
						AddRow(63, date, "go-project", 1, 1, 1, "t3g88uo"))
			},
			expectedError: nil,
			expectedProject: &warnly.Project{
				ID:        63,
				CreatedAt: date,
				Name:      "go-project",
				UserID:    1,
				TeamID:    1,
				Platform:  1,
				Key:       "t3g88uo",
			},
		},
		{
			name: "NotFound",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "name", "user_id", "team_id", "platform", "project_key"}))
			},
			expectedError:   fmt.Errorf("mysql project store: get project with id 1: %w", warnly.ErrProjectNotFound),
			expectedProject: nil,
		},
		{
			name: "QueryError",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnError(queryError)
			},
			expectedError:   fmt.Errorf("mysql project store: get project: %w", queryError),
			expectedProject: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %v", err)
			}
			defer db.Close()
			tt.mockExpect(mock)

			store := mysql.NewProjectStore(db)

			project, err := store.GetProject(t.Context(), 1)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedProject, project)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
