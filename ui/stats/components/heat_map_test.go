package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHeatMapWeekdayLabelsStartOnMonday(t *testing.T) {
	heatMap := NewHeatMap()

	require.Equal(t, "Mon │  \nTue │  \nWed │  \nThu │  \nFri │  \nSat │  \nSun │  ", heatMap.buildWeekDayLabels())
}

func TestMondayFirstWeekday(t *testing.T) {
	require.Equal(t, 0, mondayFirstWeekday(time.Monday))
	require.Equal(t, 6, mondayFirstWeekday(time.Sunday))
}
