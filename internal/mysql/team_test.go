package mysql_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vk-rv/warnly/internal/mysql"
	"github.com/vk-rv/warnly/internal/warnly"
)

func TestListTeams(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		teamStore := mysql.NewTeamStore(db)

		date := time.Date(2025, 1, 29, 6, 47, 9, 0, time.UTC)

		rows := sqlmock.NewRows([]string{"id", "created_at", "name", "owner_id"}).
			AddRow(1, date, "Team A", 1).
			AddRow(2, date, "Team B", 2)

		mock.ExpectQuery(`SELECT t.id, t.created_at, t.name, t.owner_id FROM team_relation AS tr JOIN team AS t WHERE tr.user_id = ?`).
			WithArgs(1).
			WillReturnRows(rows)

		teams, err := teamStore.ListTeams(t.Context(), 1)
		require.NoError(t, err)
		require.Len(t, teams, 2)

		expectedTeams := []warnly.Team{
			{ID: 1, CreatedAt: date, Name: "Team A", OwnerID: 1},
			{ID: 2, CreatedAt: date, Name: "Team B", OwnerID: 2},
		}

		require.ElementsMatch(t, expectedTeams, teams)
	})
}
