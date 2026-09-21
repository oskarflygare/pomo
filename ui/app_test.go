package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
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
