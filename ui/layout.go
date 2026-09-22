package ui

import (
	"fmt"
	"time"

	"github.com/Bahaaio/pomo/ui/ascii"
	"github.com/Bahaaio/pomo/ui/colors"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxWidth           = 80
	margin             = 4
	padding            = 2
	separator          = " — "
	miniatureGap       = " "
	pausedIndicator    = "(paused)"
	completedIndicator = "done!"
)

func (m *Model) buildConfirmDialogView() string {
	idle := time.Since(m.confirmStartTime).Truncate(time.Second)
	title := m.currentTaskType.Opposite().GetTask().Title

	// if we're prompting to start a long break
	if m.cyclePosition == m.longBreak.After {
		title = "long " + title
	}

	return m.confirmDialog.View("start "+title+"?", time.Duration(idle))
}

func (m *Model) buildMainContent() string {
	timeLeft := m.buildTimeLeft()

	if m.useTimerArt {
		return timeLeft + "\n\n" + m.currentTask.Title
	}

	content := m.currentTask.Title
	if !m.timer.Timedout() {
		content += separator + timeLeft
	}

	return content
}

func (m *Model) buildStatusIndicators() string {
	if m.timer.Timedout() {
		return separator + completedIndicator
	}

	indicators := ""

	if m.longBreak.Enabled {
		indicators += fmt.Sprintf(" · %d/%d", m.cyclePosition, m.longBreak.After)
	}

	if m.sessionState == Paused {
		indicators += " " + pausedIndicator
	}

	return indicators
}

func (m *Model) buildProgressBar() string {
	return "\n\n" + m.progressBar.View() + "\n"
}

func (m *Model) buildMiniatureContent() string {
	timer := m.buildPlainTimeLeft()
	if m.progressBar.Width == 0 {
		return timer
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, timer, miniatureGap, m.progressBar.View())
}

func (m *Model) updateProgressBarWidth() {
	fullBudget := max(0, min(m.width-2*padding-margin, maxWidth))
	if m.miniature {
		m.progressBar.Width = max(0, fullBudget-lipgloss.Width(m.buildPlainTimeLeft())-lipgloss.Width(miniatureGap))
		return
	}

	m.progressBar.Width = fullBudget
}

// buildPlainTimeLeft returns time left in MM:SS or HH:MM:SS format.
func (m *Model) buildPlainTimeLeft() string {
	left := m.timer.Timeout
	hours := int(left.Hours())
	minutes := int(left.Minutes()) % 60
	seconds := int(left.Seconds()) % 60

	time := ""

	// only show hours if they are non-zero
	if hours > 0 {
		time += fmt.Sprintf("%02d:", hours)
	}
	return time + fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (m *Model) buildTimeLeft() string {
	time := m.buildPlainTimeLeft()

	if m.useTimerArt {
		time = ascii.RenderNumber(time, m.timerFont)

		// remove color on pause
		if m.sessionState == Paused {
			noColor := m.asciiTimerStyle.Foreground(colors.PauseFg)
			return noColor.Render(time)
		}

		return m.asciiTimerStyle.Render(time)
	}

	return time
}

func (m *Model) buildHelpView() string {
	return m.help.View(keyMap)
}

func (m Model) buildWaitingForCommandsView() string {
	help := m.help.View(KeyMap{Quit: keyMap.Quit})

	message := lipgloss.JoinVertical(
		lipgloss.Center,
		"Waiting for post commands to complete...",
		"\n",
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		message,
	)
}
