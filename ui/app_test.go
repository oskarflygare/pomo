package ui

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestTextToggleHidesAndRestoresTimerText(t *testing.T) {
	m := Model{
		progressBar:  progress.New(),
		help:         help.New(),
		timer:        timer.New(time.Minute),
		width:        80,
		height:       24,
		currentTask:  config.Task{Title: "focus time"},
		sessionState: Paused,
		showText:     true,
		useTimerArt:  false,
	}

	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	compactView := m.View()

	assert.NotContains(t, compactView, "focus time")
	assert.NotContains(t, compactView, "pause/resume")
	assert.NotContains(t, compactView, ":")
	assert.NotContains(t, compactView, "0%")
	assert.NotContains(t, compactView, pausedIndicator)

	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	fullView := m.View()

	assert.Contains(t, fullView, "focus time")
	assert.Contains(t, fullView, "pause/resume")
	assert.Contains(t, fullView, pausedIndicator)
}
