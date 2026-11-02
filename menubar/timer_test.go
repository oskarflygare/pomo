package menubar

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var t0 = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func setupConfig(t *testing.T) {
	t.Helper()

	previous := config.C
	t.Cleanup(func() { config.C = previous })

	config.C = config.Config{
		OnSessionEnd: "ask",
		Work:         config.Task{Title: "work session", Duration: 25 * time.Minute},
		Break:        config.Task{Title: "break session", Duration: 5 * time.Minute},
		LongBreak:    config.LongBreak{Enabled: true, After: 2, Duration: 15 * time.Minute},
	}
}

func TestTimerToggleTracksWallClock(t *testing.T) {
	setupConfig(t)
	timer := NewTimer(config.WorkTask, config.C.LongBreak)

	require.False(t, timer.Running())
	assert.Equal(t, 25*time.Minute, timer.Remaining(t0))

	timer.Toggle(t0)
	assert.True(t, timer.Running())
	assert.Equal(t, 10*time.Minute, timer.Elapsed(t0.Add(10*time.Minute)))

	// pausing freezes elapsed time
	timer.Toggle(t0.Add(10 * time.Minute))
	assert.False(t, timer.Running())
	assert.Equal(t, 10*time.Minute, timer.Elapsed(t0.Add(time.Hour)))

	// resuming continues from the frozen value
	timer.Toggle(t0.Add(time.Hour))
	assert.Equal(t, 15*time.Minute, timer.Elapsed(t0.Add(time.Hour+5*time.Minute)))
	assert.InDelta(t, 0.6, timer.Progress(t0.Add(time.Hour+5*time.Minute)), 0.001)
}

func TestTimerCompletes(t *testing.T) {
	setupConfig(t)
	timer := NewTimer(config.WorkTask, config.C.LongBreak)
	timer.Toggle(t0)

	end := t0.Add(2 * time.Hour) // e.g. after system sleep
	assert.True(t, timer.Done(end))
	assert.Equal(t, 25*time.Minute, timer.Elapsed(end))
	assert.Equal(t, 1.0, timer.Progress(end))

	// a completed session cannot be resumed
	timer.Toggle(end)
	timer.Toggle(end)
	assert.False(t, timer.Running())
}

func TestTimerReset(t *testing.T) {
	setupConfig(t)
	timer := NewTimer(config.WorkTask, config.C.LongBreak)
	timer.Toggle(t0)

	now := t0.Add(10 * time.Minute)
	timer.Reset(now)

	assert.True(t, timer.Running())
	assert.Equal(t, time.Duration(0), timer.Elapsed(now))
	assert.Equal(t, time.Minute, timer.Elapsed(now.Add(time.Minute)))
}

func TestTimerSetDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantDone bool
	}{
		{"longer keeps elapsed", 50 * time.Minute, false},
		{"shorter than elapsed completes", 5 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupConfig(t)
			timer := NewTimer(config.WorkTask, config.C.LongBreak)
			timer.Toggle(t0)

			now := t0.Add(10 * time.Minute)
			timer.SetDuration(now, tt.duration)

			assert.Equal(t, tt.duration, timer.Duration())
			assert.Equal(t, tt.wantDone, timer.Done(now))
			assert.Equal(t, min(10*time.Minute, tt.duration), timer.Elapsed(now))
		})
	}
}

func TestTimerNextFollowsLongBreakCycle(t *testing.T) {
	setupConfig(t)
	timer := NewTimer(config.WorkTask, config.C.LongBreak)

	// long break after 2 work sessions
	want := []struct {
		title    string
		duration time.Duration
		long     bool
	}{
		{"break session", 5 * time.Minute, false},
		{"work session", 25 * time.Minute, false},
		{"long break session", 15 * time.Minute, true},
		{"work session", 25 * time.Minute, false},
		{"break session", 5 * time.Minute, false},
		{"work session", 25 * time.Minute, false},
		{"long break session", 15 * time.Minute, true},
	}

	for i, w := range want {
		timer.Next(t0, false)
		assert.Equal(t, w.title, timer.Task().Title, "step %d", i)
		assert.Equal(t, w.duration, timer.Duration(), "step %d", i)
		assert.Equal(t, w.long, timer.IsLongBreak(), "step %d", i)
	}
}

func TestTimerNextAutoStart(t *testing.T) {
	setupConfig(t)
	timer := NewTimer(config.WorkTask, config.C.LongBreak)
	timer.Toggle(t0)

	timer.Next(t0.Add(25*time.Minute), true)
	assert.True(t, timer.Running())
	assert.Equal(t, config.BreakTask, timer.TaskType())

	timer.Next(t0.Add(30*time.Minute), false)
	assert.False(t, timer.Running())
	assert.Equal(t, time.Duration(0), timer.Elapsed(t0.Add(40*time.Minute)))
}
