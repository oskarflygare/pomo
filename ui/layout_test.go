package ui

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestMiniatureTimerIncludesPhase(t *testing.T) {
	for _, tc := range []struct {
		name     string
		taskType config.TaskType
		title    string
		label    string
		barWidth int
	}{
		{"work with custom title", config.WorkTask, "write report", "Work", 61},
		{"break", config.BreakTask, "break session", "Break", 60},
		{"long break", config.BreakTask, "long break session", "Break", 60},
		{"short work", config.WorkTask, "short work session", "Work", 61},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := miniatureTestModel(time.Minute)
			m.currentTaskType = tc.taskType
			m.currentTask.Title = tc.title
			m.miniature = true
			m.progressBar.ShowPercentage = false
			m.handleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})

			content := m.buildMiniatureContent()
			assert.True(t, containsSameLine(content, tc.label+" 01:00", "░"), content)
			assert.Equal(t, tc.barWidth, m.progressBar.Width)
			assert.Equal(t, 72, lipgloss.Width(content))
		})
	}
}

func TestMiniatureTimerOmitsPhaseWhenTooNarrow(t *testing.T) {
	m := miniatureTestModel(time.Minute)
	m.miniature = true
	m.handleWindowResize(tea.WindowSizeMsg{Width: 5, Height: 24})
	assert.Equal(t, "01:00", m.buildMiniatureContent())
}
