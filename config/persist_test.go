package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{25 * time.Minute, "25m"},
		{time.Hour, "1h"},
		{90 * time.Minute, "1h30m"},
		{90 * time.Second, "1m30s"},
		{45 * time.Second, "45s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatDuration(tt.in))
		})
	}
}

func TestSaveDurationPreservesComments(t *testing.T) {
	sample, err := os.ReadFile(filepath.Join("..", ConfigFile))
	require.NoError(t, err)

	setupViper()
	writeAndLoadConfig(t, string(sample))
	path := viper.ConfigFileUsed()

	require.NoError(t, SaveDuration(WorkTask, 50*time.Minute))
	require.NoError(t, SaveDuration(BreakTask, 10*time.Minute))

	assert.Equal(t, 50*time.Minute, C.Work.Duration)
	assert.Equal(t, 10*time.Minute, C.Break.Duration)

	// only the duration values change; comments, blank lines and emoji stay intact
	want := strings.Replace(string(sample), "duration: 25m", "duration: 50m", 1)
	want = strings.Replace(want, "duration: 5m", "duration: 10m", 1)

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(saved))

	// the saved file loads back with the same values
	before := C
	setupViper()
	viper.SetConfigFile(path)
	require.NoError(t, LoadConfig())
	assert.Equal(t, before, C)
}

func TestSaveDurationCreatesConfigFile(t *testing.T) {
	setupViper()
	t.Setenv("HOME", t.TempDir())

	require.NoError(t, SaveDuration(BreakTask, 7*time.Minute))

	dir, err := getConfigDir()
	require.NoError(t, err)
	saved, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	require.NoError(t, err)
	assert.Equal(t, "break:\n  duration: 7m\n", string(saved))
}
