package components

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/db"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestBarChartFitsConfiguredHeightWithNoWorkSessions(t *testing.T) {
	chart := NewBarChart(12)
	stats := make([]db.DailyStat, 7)

	view := chart.View(stats)

	require.LessOrEqual(t, lipgloss.Height(view), 12)
}

func TestBarChartShowsWorkSessionCountsBelowWeekdays(t *testing.T) {
	chart := NewBarChart(12)
	stats := []db.DailyStat{
		{Date: "2026-01-05", WorkDuration: 50 * time.Minute, WorkSessions: 2},
		{Date: "2026-01-06", WorkDuration: 25 * time.Minute, WorkSessions: 1},
		{Date: "2026-01-07", WorkSessions: 0},
		{Date: "2026-01-08", WorkDuration: 75 * time.Minute, WorkSessions: 3},
		{Date: "2026-01-09", WorkDuration: 25 * time.Minute, WorkSessions: 1},
		{Date: "2026-01-10", WorkSessions: 0},
		{Date: "2026-01-11", WorkDuration: 25 * time.Minute, WorkSessions: 1},
	}

	view := chart.View(stats)

	require.Contains(t, view, "Mon  Tue  Wed  Thu  Fri  Sat  Sun  \n          2x   1x   0x   3x   1x   0x   1x")
}
