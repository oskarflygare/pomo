//go:build unix

package menubar

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "menubar.lock")

	first, err := AcquireInstanceLock(path)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(os.Getpid())+"\n", string(content))

	// a second lock is refused while the first is held, and names its pid
	_, err = AcquireInstanceLock(path)
	var running *AlreadyRunningError
	require.ErrorAs(t, err, &running)
	assert.Equal(t, os.Getpid(), running.PID)

	// once released, the lock can be taken again
	require.NoError(t, first.Release())
	assert.NoError(t, first.Release(), "releasing twice is fine")

	second, err := AcquireInstanceLock(path)
	require.NoError(t, err)
	assert.NoError(t, second.Release())
}
