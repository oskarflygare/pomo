package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/ui/ascii"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestTextToggleShowsAndRestoresMiniatureTimerLayout(t *testing.T) {
	m := Model{
		progressBar:  progress.New(),
		help:         help.New(),
		timer:        timer.New(time.Minute),
		width:        80,
		height:       24,
		currentTask:  config.Task{Title: "focus time"},
		sessionState: Paused,
		useTimerArt:  false,
	}

	m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	miniatureView := m.View()

	assert.NotContains(t, miniatureView, "focus time")
	assert.NotContains(t, miniatureView, "pause/resume")
	assert.NotContains(t, miniatureView, "0%")
	assert.NotContains(t, miniatureView, pausedIndicator)
	assert.True(t, containsSameLine(miniatureView, "01:00", "░"))

	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	fullView := m.View()

	assert.Contains(t, fullView, "focus time")
	assert.Contains(t, fullView, "pause/resume")
	assert.Contains(t, fullView, "0%")
	assert.Contains(t, fullView, pausedIndicator)
}

func containsSameLine(view string, substrings ...string) bool {
	for _, line := range strings.Split(view, "\n") {
		containsAll := true
		for _, substring := range substrings {
			containsAll = containsAll && strings.Contains(line, substring)
		}
		if containsAll {
			return true
		}
	}

	return false
}

func TestMiniatureLayoutLeavesConfirmationAndWaitingStatesUnchanged(t *testing.T) {
	for _, sessionState := range []SessionState{ShowingConfirm, WaitingForCommands} {
		t.Run(sessionStateName(sessionState), func(t *testing.T) {
			m := miniatureTestModel(time.Minute)
			m.sessionState = sessionState

			m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

			assert.False(t, m.miniature)
			assert.True(t, m.progressBar.ShowPercentage)
		})
	}
}

func TestMiniatureLayoutPersistsAcrossSessionTransitions(t *testing.T) {
	originalConfig := config.C
	config.C = config.Config{
		Work:  config.Task{Title: "work", Duration: 2 * time.Hour},
		Break: config.Task{Title: "break", Duration: 5 * time.Minute},
	}
	t.Cleanup(func() { config.C = originalConfig })

	m := miniatureTestModel(config.C.Work.Duration)
	m.currentTaskType = config.WorkTask
	m.currentTask = config.C.Work
	m.longBreak = config.LongBreak{Enabled: true, After: 4, Duration: 15 * time.Minute}
	m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	m.nextSession()
	assertMiniatureState(t, m, "break")
	m.nextSession()
	assertMiniatureState(t, m, "work")

	m.cyclePosition = 1
	m.longBreak = config.LongBreak{Enabled: true, After: 1, Duration: 15 * time.Minute}
	m.nextSession()
	assertMiniatureState(t, m, "long break")
	m.shortSession()
	assertMiniatureState(t, m, "short break")
}

func TestMiniatureLayoutRecalculatesProgressWidth(t *testing.T) {
	t.Run("key handlers", func(t *testing.T) {
		m := miniatureTestModel(59 * time.Minute)
		m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		assert.Equal(t, miniatureProgressWidth(m), m.progressBar.Width)

		m.currentTask.Duration = 2 * time.Hour
		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
		assert.Equal(t, miniatureProgressWidth(m), m.progressBar.Width)
	})

	t.Run("timer tick", func(t *testing.T) {
		m := miniatureTestModel(time.Hour)
		m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

		m.handleTimerTick(timer.TickMsg{})
		assert.Equal(t, miniatureProgressWidth(m), m.progressBar.Width)
	})

	t.Run("narrow terminal", func(t *testing.T) {
		m := miniatureTestModel(time.Minute)
		m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m.handleWindowResize(tea.WindowSizeMsg{Width: 5, Height: 24})

		assert.Equal(t, 0, m.progressBar.Width)
		assert.True(t, containsSameLine(m.View(), "01:00"))
		assert.NotContains(t, m.View(), "░")

		m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		assert.Equal(t, 0, m.progressBar.Width)
	})
}

func TestMiniatureLayoutUsesPlainTimerWhenASCIIArtIsEnabled(t *testing.T) {
	for _, fontName := range []string{ascii.Mono12, ascii.Rebel, ascii.Ansi, ascii.AnsiShadow} {
		t.Run(fontName, func(t *testing.T) {
			m := miniatureTestModel(time.Minute)
			m.useTimerArt = true
			m.timerFont = ascii.GetFont(fontName)
			m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
			m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

			content := m.buildMiniatureContent()

			assert.Equal(t, 1, lipgloss.Height(content))
			assert.True(t, containsSameLine(content, "01:00", "░"))
			assert.Equal(t, 66, m.progressBar.Width)
		})
	}
}

func miniatureTestModel(duration time.Duration) Model {
	return Model{
		progressBar:   progress.New(),
		help:          help.New(),
		timer:         timer.New(duration),
		duration:      duration,
		width:         80,
		height:        24,
		currentTask:   config.Task{Title: "focus time", Duration: duration},
		cyclePosition: 1,
		sessionState:  Running,
	}
}

func miniatureProgressWidth(m Model) int {
	fullBudget := max(0, min(m.width-2*padding-margin, maxWidth))
	return max(0, fullBudget-lipgloss.Width(m.buildPlainTimeLeft())-lipgloss.Width(miniatureGap))
}

func assertMiniatureState(t *testing.T, m Model, title string) {
	t.Helper()
	assert.True(t, m.miniature)
	assert.False(t, m.progressBar.ShowPercentage)
	assert.Equal(t, title, m.currentTask.Title)
	assert.Equal(t, miniatureProgressWidth(m), m.progressBar.Width)
}

func sessionStateName(state SessionState) string {
	if state == ShowingConfirm {
		return "confirmation"
	}
	return "waiting"
}
