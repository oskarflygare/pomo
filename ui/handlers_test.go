package ui

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/stretchr/testify/assert"
)

func TestHandleCompletionAutoStartBreaks(t *testing.T) {
	tests := []struct {
		name            string
		onSessionEnd    string
		autoStartBreaks bool
		completed       config.TaskType
		wantType        config.TaskType
		wantState       SessionState
	}{
		{"work ends, auto-start on", "ask", true, config.WorkTask, config.BreakTask, Running},
		{"work ends, auto-start overrides quit", "quit", true, config.WorkTask, config.BreakTask, Running},
		{"work ends, auto-start off", "ask", false, config.WorkTask, config.WorkTask, ShowingConfirm},
		{"break ends, auto-start on still asks", "ask", true, config.BreakTask, config.BreakTask, ShowingConfirm},
		{"break ends with onSessionEnd start", "start", false, config.BreakTask, config.WorkTask, Running},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := config.C
			config.C = config.Config{
				Work:  config.Task{Title: "work", Duration: 25 * time.Minute},
				Break: config.Task{Title: "break", Duration: 5 * time.Minute},
			}
			t.Cleanup(func() { config.C = originalConfig })

			m := miniatureTestModel(tt.completed.GetTask().Duration)
			m.onSessionEnd = tt.onSessionEnd
			m.autoStartBreaks = tt.autoStartBreaks
			m.currentTaskType = tt.completed
			m.currentTask = *tt.completed.GetTask()
			m.sessionState = Running

			m.handleCompletion()

			assert.Equal(t, tt.wantType, m.currentTaskType)
			assert.Equal(t, tt.wantState, m.sessionState)
			if tt.wantState == Running {
				assert.Nil(t, m.commandsWg, "auto-started sessions leave post actions running in the background")
			}
		})
	}
}
