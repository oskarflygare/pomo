// Package menubar implements the macOS menu bar timer:
// a wall-clock pomodoro engine, a controller that builds the
// status label and menu, and a native status item (darwin only).
package menubar

import (
	"time"

	"github.com/Bahaaio/pomo/config"
)

// Timer is a pomodoro session engine driven by wall-clock time,
// so it stays correct across missed ticks and system sleep.
type Timer struct {
	taskType      config.TaskType
	task          config.Task
	isLongBreak   bool
	longBreak     config.LongBreak
	cyclePosition int // for long break tracking

	elapsed   time.Duration // accumulated before the current run
	startedAt time.Time     // zero while paused
}

// NewTimer returns a paused timer at the start of a session of the given type.
func NewTimer(taskType config.TaskType, longBreak config.LongBreak) *Timer {
	return &Timer{
		taskType:      taskType,
		task:          *taskType.GetTask(),
		longBreak:     longBreak,
		cyclePosition: 1,
	}
}

func (t *Timer) TaskType() config.TaskType { return t.taskType }
func (t *Timer) Task() config.Task         { return t.task }
func (t *Timer) IsLongBreak() bool         { return t.isLongBreak }
func (t *Timer) Running() bool             { return !t.startedAt.IsZero() }
func (t *Timer) Duration() time.Duration   { return t.task.Duration }

// Elapsed returns the time spent in the current session, capped at its duration.
func (t *Timer) Elapsed(now time.Time) time.Duration {
	elapsed := t.elapsed
	if t.Running() {
		elapsed += now.Sub(t.startedAt)
	}

	return min(max(elapsed, 0), t.task.Duration)
}

func (t *Timer) Remaining(now time.Time) time.Duration {
	return t.task.Duration - t.Elapsed(now)
}

// Progress returns the completed fraction of the current session in [0, 1].
func (t *Timer) Progress(now time.Time) float64 {
	if t.task.Duration <= 0 {
		return 1
	}

	return float64(t.Elapsed(now)) / float64(t.task.Duration)
}

func (t *Timer) Done(now time.Time) bool {
	return t.Remaining(now) <= 0
}

// Toggle starts or pauses the timer. A completed session cannot be resumed.
func (t *Timer) Toggle(now time.Time) {
	if t.Running() {
		t.elapsed = t.Elapsed(now)
		t.startedAt = time.Time{}
		return
	}

	if !t.Done(now) {
		t.startedAt = now
	}
}

// Reset restarts the current session from zero, keeping its running state.
func (t *Timer) Reset(now time.Time) {
	t.elapsed = 0
	if t.Running() {
		t.startedAt = now
	}
}

// SetDuration changes the duration of the current session.
// Time already spent is kept, so a shorter duration may complete the session.
func (t *Timer) SetDuration(now time.Time, duration time.Duration) {
	elapsed := t.elapsed
	if t.Running() {
		elapsed += now.Sub(t.startedAt)
		t.startedAt = now
	}

	t.task.Duration = duration
	t.elapsed = min(elapsed, duration)
}

// Next moves to the following session (work <-> break),
// starting a long break when the cycle calls for one.
// The new session starts running if autoStart is true.
func (t *Timer) Next(now time.Time, autoStart bool) {
	nextType := t.taskType.Opposite()
	next := *nextType.GetTask()
	isLongBreak := false

	if t.longBreak.Enabled {
		// increment step count after break sessions
		if t.taskType == config.BreakTask {
			t.cyclePosition++
		}

		if t.taskType == config.WorkTask && t.cyclePosition == t.longBreak.After {
			next.Duration = t.longBreak.Duration
			next.Title = "long " + next.Title
			isLongBreak = true
		}

		// reset step count after long break
		if t.cyclePosition > t.longBreak.After {
			t.cyclePosition = 1
		}
	}

	t.taskType = nextType
	t.task = next
	t.isLongBreak = isLongBreak
	t.elapsed = 0
	t.startedAt = time.Time{}

	if autoStart {
		t.startedAt = now
	}
}
