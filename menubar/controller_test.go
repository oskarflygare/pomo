package menubar

import (
	"errors"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedSession struct {
	duration    time.Duration
	sessionType db.SessionType
}

type fakeRepo struct {
	sessions []recordedSession
	err      error
}

func (r *fakeRepo) CreateSession(_ time.Time, d time.Duration, st db.SessionType) error {
	r.sessions = append(r.sessions, recordedSession{d, st})
	return nil
}

func (r *fakeRepo) GetWeeklyStats() ([]db.DailyStat, error) {
	return []db.DailyStat{
		{Date: "2025-12-31", WorkDuration: 2 * time.Hour, WorkSessions: 4},
		{Date: "2026-01-01", WorkDuration: 75 * time.Minute, WorkSessions: 3},
	}, r.err
}

func (r *fakeRepo) GetStreakStats() (db.StreakStats, error) {
	return db.StreakStats{Current: 1, Best: 12}, r.err
}

func (r *fakeRepo) GetAllTimeStats() (db.AllTimeStats, error) {
	return db.AllTimeStats{TotalWorkDuration: 52 * time.Hour}, r.err
}

type fakeLogin struct {
	enabled bool
	err     error
}

func (l *fakeLogin) Enabled() bool { return l.enabled }

func (l *fakeLogin) Enable() error {
	if l.err == nil {
		l.enabled = true
	}
	return l.err
}

func (l *fakeLogin) Disable() error {
	if l.err == nil {
		l.enabled = false
	}
	return l.err
}

type fakeView struct{ status Status }

func (v *fakeView) SetStatus(s Status) { v.status = s }

type harness struct {
	*Controller
	clock   time.Time
	repo    *fakeRepo
	view    *fakeView
	login   *fakeLogin
	actions []string
	saved   map[config.TaskType]time.Duration
	saveErr error
	quit    bool
}

func newHarness(t *testing.T, onSessionEnd string) *harness {
	t.Helper()
	setupConfig(t)
	config.C.OnSessionEnd = onSessionEnd

	h := &harness{clock: t0, repo: &fakeRepo{}, view: &fakeView{}, login: &fakeLogin{}, saved: map[config.TaskType]time.Duration{}}
	h.Controller = NewController(config.C, h.repo, h.view, h.login, func() { h.quit = true })
	h.now = func() time.Time { return h.clock }
	h.runActions = func(task config.Task) { h.actions = append(h.actions, task.Title) }
	h.saveDuration = func(tt config.TaskType, d time.Duration) error {
		h.saved[tt] = d
		tt.GetTask().Duration = d
		return nil
	}
	h.saveAutoStartBreaks = func(bool) error { return h.saveErr }

	return h
}

func (h *harness) advance(d time.Duration) {
	h.clock = h.clock.Add(d)
	h.Tick()
}

func TestControllerStatus(t *testing.T) {
	h := newHarness(t, "ask")

	h.Tick()
	assert.Equal(t, Status{Label: "Work 25:00", Progress: 0, Paused: true}, h.view.status)

	h.Toggle()
	h.advance(21*time.Minute + 50*time.Second)
	assert.Equal(t, "Work 03:10", h.view.status.Label)
	assert.False(t, h.view.status.Paused)

	// partial seconds round up
	h.advance(500 * time.Millisecond)
	assert.Equal(t, "Work 03:10", h.view.status.Label)
}

func TestControllerCompletion(t *testing.T) {
	tests := []struct {
		onSessionEnd string
		wantRunning  bool
	}{
		{"ask", false},
		{"quit", false},
		{"start", true},
	}

	for _, tt := range tests {
		t.Run(tt.onSessionEnd, func(t *testing.T) {
			h := newHarness(t, tt.onSessionEnd)
			h.Toggle()
			h.advance(25 * time.Minute)

			assert.Equal(t, []recordedSession{{25 * time.Minute, db.WorkSession}}, h.repo.sessions)
			assert.Equal(t, []string{"work session"}, h.actions)
			assert.Equal(t, "Break 05:00", h.view.status.Label)
			assert.Equal(t, !tt.wantRunning, h.view.status.Paused)
			assert.False(t, h.quit, "the menu bar never quits on its own")
		})
	}
}

func TestControllerSkipRecordsElapsed(t *testing.T) {
	h := newHarness(t, "ask")
	h.Toggle()
	h.advance(10 * time.Minute)

	h.Skip()
	assert.Equal(t, []recordedSession{{10 * time.Minute, db.WorkSession}}, h.repo.sessions)
	assert.Empty(t, h.actions, "skipping does not notify")
	assert.Equal(t, "Break 05:00", h.view.status.Label)
	assert.False(t, h.view.status.Paused)

	// skipping an unstarted session records nothing
	h.Toggle()
	h.Skip()
	assert.Len(t, h.repo.sessions, 1)
}

func TestControllerQuitRecordsElapsed(t *testing.T) {
	h := newHarness(t, "ask")
	h.Toggle()
	h.advance(90 * time.Second)

	h.Quit()
	assert.True(t, h.quit)
	assert.Equal(t, []recordedSession{{90 * time.Second, db.WorkSession}}, h.repo.sessions)
}

func TestControllerSetDuration(t *testing.T) {
	h := newHarness(t, "ask")
	h.Toggle()
	h.advance(10 * time.Minute)

	// changing the other session type leaves the current one alone
	h.SetDuration(config.BreakTask, 10*time.Minute)
	assert.Equal(t, "Work 15:00", h.view.status.Label)

	h.SetDuration(config.WorkTask, 50*time.Minute)
	assert.Equal(t, "Work 40:00", h.view.status.Label)
	assert.Equal(t, map[config.TaskType]time.Duration{config.WorkTask: 50 * time.Minute, config.BreakTask: 10 * time.Minute}, h.saved)

	// shortening below the elapsed time completes the session
	h.SetDuration(config.WorkTask, 5*time.Minute)
	assert.Equal(t, "Break 10:00", h.view.status.Label)
	assert.Equal(t, []string{"work session"}, h.actions)
}

func TestControllerSetDurationSaveError(t *testing.T) {
	h := newHarness(t, "ask")
	h.saveDuration = func(config.TaskType, time.Duration) error { return errors.New("read-only") }

	h.SetDuration(config.WorkTask, 45*time.Minute)
	assert.Equal(t, 45*time.Minute, config.C.Work.Duration, "applies in memory anyway")
	assert.Equal(t, "Work 45:00", h.view.status.Label)
	assert.Contains(t, titles(h.Menu()), "Could not save setting to pomo.yaml")
}

func TestControllerMenu(t *testing.T) {
	h := newHarness(t, "ask")
	menu := h.Menu()
	require.Len(t, menu, 16)

	assert.Equal(t, []string{
		"Work · 25:00 left", "Start", "Skip to Break", "Reset", "",
		"Today", "Last 7 days", "Streak", "All time", "",
		"Work length", "Break length", "Auto-start Breaks", "",
		"Open at Login", "Quit pomo",
	}, titles(menu))

	assert.Equal(t, "3 sessions · 1h 15m", detail(menu, "Today"))
	assert.Equal(t, "3h 15m", detail(menu, "Last 7 days"))
	assert.Equal(t, "1 day · best 12", detail(menu, "Streak"))
	assert.Equal(t, "52h focused", detail(menu, "All time"))
	assert.Equal(t, "25m", detail(menu, "Work length"))
	assert.False(t, find(menu, "Reset").Enabled)

	// choosing a preset applies it
	work := find(menu, "Work length").Submenu
	assert.True(t, find(work, "25m").Checked)
	find(work, "45m").Action()
	assert.Equal(t, "Work 45:00", h.view.status.Label)

	h.Toggle()
	h.advance(time.Minute)
	h.Toggle()
	menu = h.Menu()
	assert.Equal(t, "Work · 44:00 left (paused)", menu[0].Title)
	assert.Equal(t, "Resume", menu[1].Title)
	assert.True(t, find(menu, "Reset").Enabled)
}

func TestControllerAutoStartBreaks(t *testing.T) {
	h := newHarness(t, "ask")
	assert.False(t, find(h.Menu(), "Auto-start Breaks").Checked)

	find(h.Menu(), "Auto-start Breaks").Action()
	assert.True(t, find(h.Menu(), "Auto-start Breaks").Checked)

	// work completion starts the break
	h.Toggle()
	h.advance(25 * time.Minute)
	assert.Equal(t, "Break 05:00", h.view.status.Label)
	assert.False(t, h.view.status.Paused)

	// break completion still waits for a click
	h.advance(5 * time.Minute)
	assert.Equal(t, "Work 25:00", h.view.status.Label)
	assert.True(t, h.view.status.Paused)

	// switching it off again restores waiting after work
	find(h.Menu(), "Auto-start Breaks").Action()
	h.Toggle()
	h.advance(25 * time.Minute)
	assert.Equal(t, "Break 15:00", h.view.status.Label, "second cycle is a long break")
	assert.True(t, h.view.status.Paused)
}

func TestControllerAutoStartBreaksSaveError(t *testing.T) {
	h := newHarness(t, "ask")
	h.saveErr = errors.New("read-only")

	find(h.Menu(), "Auto-start Breaks").Action()

	menu := h.Menu()
	assert.True(t, find(menu, "Auto-start Breaks").Checked, "applies in memory anyway")
	assert.Contains(t, titles(menu), "Could not save setting to pomo.yaml")
}

func TestControllerAutoStartBreaksLockedByOnSessionEnd(t *testing.T) {
	h := newHarness(t, "start")

	item := find(h.Menu(), "Auto-start Breaks")
	assert.True(t, item.Checked)
	assert.False(t, item.Enabled)
	assert.Equal(t, "onSessionEnd: start", item.Detail)
}

func TestControllerMenuCustomDuration(t *testing.T) {
	h := newHarness(t, "ask")
	config.C.Break.Duration = 7 * time.Minute

	breaks := find(h.Menu(), "Break length").Submenu
	assert.Equal(t, []string{"3m", "5m", "7m", "10m", "15m", "20m"}, titles(breaks))
	assert.True(t, find(breaks, "7m").Checked)
}

func TestControllerMenuWithoutStats(t *testing.T) {
	tests := []struct {
		name string
		repo Repo
	}{
		{"no database", nil},
		{"query error", &fakeRepo{err: errors.New("boom")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, "ask")
			h.repo = nil
			h.Controller.repo = tt.repo

			assert.Contains(t, titles(h.Menu()), "Stats unavailable")
		})
	}
}

func TestFormatClock(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "00:00"},
		{190 * time.Second, "03:10"},
		{100*time.Millisecond + 59*time.Second, "01:00"},
		{90 * time.Minute, "1:30:00"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, formatClock(tt.in))
	}
}

func titles(items []MenuItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Title
	}
	return out
}

func find(items []MenuItem, title string) MenuItem {
	for _, item := range items {
		if item.Title == title {
			return item
		}
	}
	return MenuItem{}
}

func detail(items []MenuItem, title string) string {
	return find(items, title).Detail
}

func TestControllerToggleLogin(t *testing.T) {
	h := newHarness(t, "ask")
	assert.False(t, find(h.Menu(), "Open at Login").Checked)

	find(h.Menu(), "Open at Login").Action()
	assert.True(t, h.login.enabled)
	assert.True(t, find(h.Menu(), "Open at Login").Checked)

	find(h.Menu(), "Open at Login").Action()
	assert.False(t, h.login.enabled)

	h.login.err = errors.New("permission denied")
	find(h.Menu(), "Open at Login").Action()
	assert.Contains(t, titles(h.Menu()), "Could not update login item")
}

func TestControllerMenuWithoutLogin(t *testing.T) {
	h := newHarness(t, "ask")
	h.Controller.login = nil

	assert.NotContains(t, titles(h.Menu()), "Open at Login")
}
