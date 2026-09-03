package db

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestGetDailyStatsIncludesWorkSessionCount(t *testing.T) {
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	_, err = database.Exec(schema)
	require.NoError(t, err)

	repo := NewSessionRepo(database)
	dayOne := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
	dayTwo := dayOne.AddDate(0, 0, 1)

	require.NoError(t, repo.CreateSession(dayOne, 25*time.Minute, WorkSession))
	require.NoError(t, repo.CreateSession(dayOne.Add(time.Hour), 20*time.Minute, WorkSession))
	require.NoError(t, repo.CreateSession(dayOne.Add(2*time.Hour), 5*time.Minute, BreakSession))
	require.NoError(t, repo.CreateSession(dayTwo, 10*time.Minute, BreakSession))

	stats, err := repo.getDailyStats(dayOne, dayTwo.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, []DailyStat{
		{Date: "2026-01-05", WorkDuration: 45 * time.Minute, WorkSessions: 2},
		{Date: "2026-01-06", WorkDuration: 0, WorkSessions: 0},
		{Date: "2026-01-07", WorkDuration: 0, WorkSessions: 0},
	}, stats)
}
