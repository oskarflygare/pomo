package menubar

import (
	"context"
	"fmt"
	"log"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/Bahaaio/pomo/actions"
	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
)

var (
	workPresets  = []time.Duration{15 * time.Minute, 20 * time.Minute, 25 * time.Minute, 30 * time.Minute, 45 * time.Minute, 50 * time.Minute, time.Hour, 90 * time.Minute}
	breakPresets = []time.Duration{3 * time.Minute, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 20 * time.Minute}
)

// Status is what the menu bar item displays.
type Status struct {
	Label    string  // e.g. "Work 03:10"
	Progress float64 // in [0, 1]
	Paused   bool
}

// MenuItem describes one entry of the click menu.
type MenuItem struct {
	Title     string
	Detail    string // right-aligned secondary text
	Enabled   bool
	Checked   bool
	Separator bool
	Action    func()
	Submenu   []MenuItem
}

// View renders the menu bar item.
type View interface {
	SetStatus(Status)
}

// Repo is the subset of [db.SessionRepo] used by the menu bar.
type Repo interface {
	CreateSession(startedAt time.Time, duration time.Duration, sessionType db.SessionType) error
	GetWeeklyStats() ([]db.DailyStat, error)
	GetStreakStats() (db.StreakStats, error)
	GetAllTimeStats() (db.AllTimeStats, error)
}

// LoginToggle switches starting the menu bar at login.
type LoginToggle interface {
	Enabled() bool
	Enable() error
	Disable() error
}

// Controller owns the timer and translates user actions into state changes.
// Its methods are safe to call from the UI thread and the ticker concurrently.
type Controller struct {
	mu              sync.Mutex
	timer           *Timer
	repo            Repo // nil if the database is unavailable
	view            View
	login           LoginToggle // nil hides the login menu item
	onSessionEnd    string
	autoStartBreaks bool
	menuError       string // last failed menu action, shown in the menu

	now                 func() time.Time
	saveDuration        func(config.TaskType, time.Duration) error
	saveAutoStartBreaks func(bool) error
	runActions          func(config.Task)
	quit                func()
}

// NewController returns a controller with a paused work session.
// repo and login may be nil; quit is called after the current session is recorded.
func NewController(cfg config.Config, repo Repo, view View, login LoginToggle, quit func()) *Controller {
	return &Controller{
		timer:               NewTimer(config.WorkTask, cfg.LongBreak),
		repo:                repo,
		view:                view,
		login:               login,
		onSessionEnd:        cfg.OnSessionEnd,
		autoStartBreaks:     cfg.AutoStartBreaks,
		now:                 time.Now,
		saveDuration:        config.SaveDuration,
		saveAutoStartBreaks: config.SaveAutoStartBreaks,
		runActions:          runPostActions,
		quit:                quit,
	}
}

// Tick completes the session if its time is up and refreshes the view.
func (c *Controller) Tick() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.timer.Done(now) {
		c.complete(now)
	}

	c.render(now)
}

// Toggle starts or pauses the current session.
func (c *Controller) Toggle() {
	c.update(func(now time.Time) { c.timer.Toggle(now) })
}

// Skip records the current session and moves to the next one,
// which runs if the skipped session was running.
func (c *Controller) Skip() {
	c.update(func(now time.Time) {
		running := c.timer.Running()
		c.record(now)
		c.timer.Next(now, running)
	})
}

// Reset restarts the current session from zero.
func (c *Controller) Reset() {
	c.update(func(now time.Time) { c.timer.Reset(now) })
}

// SetDuration changes and persists the length of work or break sessions,
// applying it to the current session if it is of the same type.
func (c *Controller) SetDuration(taskType config.TaskType, duration time.Duration) {
	c.update(func(now time.Time) {
		c.menuError = ""
		if err := c.saveDuration(taskType, duration); err != nil {
			log.Printf("failed to save %v duration: %v", taskType, err)
			c.menuError = "Could not save setting to pomo.yaml"
			taskType.GetTask().Duration = duration
		}

		if c.timer.TaskType() == taskType && !c.timer.IsLongBreak() {
			c.timer.SetDuration(now, duration)
		}

		if c.timer.Done(now) {
			c.complete(now)
		}
	})
}

// ToggleAutoStartBreaks switches starting breaks automatically
// when a work session ends, and persists the choice.
func (c *Controller) ToggleAutoStartBreaks() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.autoStartBreaks = !c.autoStartBreaks

	c.menuError = ""
	if err := c.saveAutoStartBreaks(c.autoStartBreaks); err != nil {
		log.Printf("failed to save autoStartBreaks: %v", err)
		c.menuError = "Could not save setting to pomo.yaml"
	}
}

// Quit records the current session and exits.
func (c *Controller) Quit() {
	c.mu.Lock()
	c.record(c.now())
	c.mu.Unlock()

	c.quit()
}

// Status returns the current menu bar status.
func (c *Controller) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.status(c.now())
}

// Menu builds the click menu for the current state.
func (c *Controller) Menu() []MenuItem {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	t := c.timer

	header := fmt.Sprintf("%s · %s left", sessionName(t), formatClock(t.Remaining(now)))
	if !t.Running() && t.Elapsed(now) > 0 {
		header += " (paused)"
	}

	toggle := "Start"
	if t.Running() {
		toggle = "Pause"
	} else if t.Elapsed(now) > 0 {
		toggle = "Resume"
	}

	skip := "Skip to Break"
	if t.TaskType() == config.BreakTask {
		skip = "Skip to Work"
	}

	items := []MenuItem{
		{Title: header},
		{Title: toggle, Detail: "right-click", Enabled: true, Action: c.Toggle},
		{Title: skip, Enabled: true, Action: c.Skip},
		{Title: "Reset", Enabled: t.Elapsed(now) > 0, Action: c.Reset},
		{Separator: true},
	}

	items = append(items, c.statsItems()...)
	items = append(items,
		MenuItem{Separator: true},
		c.durationItem("Work length", config.WorkTask, workPresets),
		c.durationItem("Break length", config.BreakTask, breakPresets),
		c.autoStartBreaksItem(),
	)

	items = append(items, MenuItem{Separator: true})

	if c.login != nil {
		items = append(items, MenuItem{
			Title:   "Open at Login",
			Enabled: true,
			Checked: c.login.Enabled(),
			Action:  c.ToggleLogin,
		})
	}

	if c.menuError != "" {
		items = append(items, MenuItem{Title: c.menuError})
	}

	return append(items, MenuItem{Title: "Quit pomo", Enabled: true, Action: c.Quit})
}

// ToggleLogin enables or disables starting the menu bar at login.
func (c *Controller) ToggleLogin() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	if c.login.Enabled() {
		err = c.login.Disable()
	} else {
		err = c.login.Enable()
	}

	c.menuError = ""
	if err != nil {
		log.Printf("failed to update login item: %v", err)
		c.menuError = "Could not update login item"
	}
}

// applies a state change and refreshes the view
func (c *Controller) update(change func(now time.Time)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	change(now)
	c.render(now)
}

func (c *Controller) render(now time.Time) {
	if c.view != nil {
		c.view.SetStatus(c.status(now))
	}
}

func (c *Controller) status(now time.Time) Status {
	phase := "Work"
	if c.timer.TaskType() == config.BreakTask {
		phase = "Break"
	}

	return Status{
		Label:    phase + " " + formatClock(c.timer.Remaining(now)),
		Progress: c.timer.Progress(now),
		Paused:   !c.timer.Running(),
	}
}

// records the finished session, runs post actions, and moves on
func (c *Controller) complete(now time.Time) {
	log.Println("menubar session completed")

	c.record(now)
	c.runActions(c.timer.Task())
	c.timer.Next(now, c.autoStartsNext())
}

// reports whether the session following the current one starts without a click
func (c *Controller) autoStartsNext() bool {
	return c.onSessionEnd == "start" ||
		(c.autoStartBreaks && c.timer.TaskType() == config.WorkTask)
}

// records the current session in the database, ignoring near-zero sessions
func (c *Controller) record(now time.Time) {
	elapsed := c.timer.Elapsed(now)
	if elapsed < time.Second || c.repo == nil {
		return
	}

	if err := c.repo.CreateSession(now, elapsed, db.GetSessionType(c.timer.TaskType())); err != nil {
		log.Printf("failed to record session: %v", err)
	}
}

func (c *Controller) statsItems() []MenuItem {
	unavailable := []MenuItem{{Title: "Stats unavailable"}}
	if c.repo == nil {
		return unavailable
	}

	weekly, err := c.repo.GetWeeklyStats()
	if err != nil || len(weekly) == 0 {
		return unavailable
	}

	streak, err := c.repo.GetStreakStats()
	if err != nil {
		return unavailable
	}

	allTime, err := c.repo.GetAllTimeStats()
	if err != nil {
		return unavailable
	}

	today := weekly[len(weekly)-1]
	var week time.Duration
	for _, day := range weekly {
		week += day.WorkDuration
	}

	return []MenuItem{
		{Title: "Today", Detail: fmt.Sprintf("%s · %s", plural(today.WorkSessions, "session"), formatHours(today.WorkDuration))},
		{Title: "Last 7 days", Detail: formatHours(week)},
		{Title: "Streak", Detail: fmt.Sprintf("%s · best %d", plural(streak.Current, "day"), streak.Best)},
		{Title: "All time", Detail: formatHours(allTime.TotalWorkDuration) + " focused"},
	}
}

func (c *Controller) durationItem(title string, taskType config.TaskType, presets []time.Duration) MenuItem {
	current := taskType.GetTask().Duration

	if !slices.Contains(presets, current) {
		presets = append(slices.Clone(presets), current)
		slices.Sort(presets)
	}

	submenu := make([]MenuItem, 0, len(presets))
	for _, d := range presets {
		submenu = append(submenu, MenuItem{
			Title:   config.FormatDuration(d),
			Enabled: true,
			Checked: d == current,
			Action:  func() { c.SetDuration(taskType, d) },
		})
	}

	return MenuItem{
		Title:   title,
		Detail:  config.FormatDuration(current),
		Enabled: true,
		Submenu: submenu,
	}
}

// with onSessionEnd: start every session auto-starts, so the toggle is shown checked and locked
func (c *Controller) autoStartBreaksItem() MenuItem {
	item := MenuItem{
		Title:   "Auto-start Breaks",
		Enabled: true,
		Checked: c.autoStartBreaks,
		Action:  c.ToggleAutoStartBreaks,
	}

	if c.onSessionEnd == "start" {
		item.Enabled = false
		item.Checked = true
		item.Detail = "onSessionEnd: start"
	}

	return item
}

func sessionName(t *Timer) string {
	switch {
	case t.IsLongBreak():
		return "Long break"
	case t.TaskType() == config.BreakTask:
		return "Break"
	default:
		return "Work"
	}
}

func runPostActions(task config.Task) {
	ctx, cancel := context.WithTimeout(context.Background(), actions.CommandTimeout)
	wg := actions.RunPostActions(ctx, task)

	go func() {
		wg.Wait()
		cancel()
	}()
}

// formats remaining time as mm:ss or h:mm:ss, rounding up to whole seconds
func formatClock(d time.Duration) string {
	seconds := int(math.Ceil(d.Seconds()))
	h, m, s := seconds/3600, seconds/60%60, seconds%60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// formats a total as e.g. "45m" or "3h 20m"
func formatHours(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
