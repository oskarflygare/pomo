package menubar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLoginItem(t *testing.T) LoginItem {
	t.Helper()
	dir := t.TempDir()

	return LoginItem{
		PlistPath:  filepath.Join(dir, "LaunchAgents", LoginLabel+".plist"),
		Executable: "/Users/me/.local/bin/pomo",
		Home:       "/Users/me",
		PathEnv:    "/opt/homebrew/bin:/usr/bin:/bin",
	}
}

func TestLoginItemEnableDisable(t *testing.T) {
	login := testLoginItem(t)
	require.False(t, login.Enabled())

	require.NoError(t, login.Enable())
	assert.True(t, login.Enabled())

	plist, err := os.ReadFile(login.PlistPath)
	require.NoError(t, err)
	for _, want := range []string{
		"<string>" + LoginLabel + "</string>",
		"<string>/Users/me/.local/bin/pomo</string>\n\t\t<string>menubar</string>\n\t\t<string>--foreground</string>",
		"<key>WorkingDirectory</key>\n\t<string>/Users/me</string>",
		"<string>/opt/homebrew/bin:/usr/bin:/bin</string>",
		"<key>RunAtLoad</key>\n\t<true/>",
		"<key>SuccessfulExit</key>\n\t\t<false/>",
	} {
		assert.Contains(t, string(plist), want)
	}

	require.NoError(t, login.Disable())
	assert.False(t, login.Enabled())

	// disabling twice is fine
	assert.NoError(t, login.Disable())
}

func TestLoginItemEscapesPaths(t *testing.T) {
	login := testLoginItem(t)
	login.Executable = "/Users/me/R&D <tools>/pomo"

	plist, err := login.plist()
	require.NoError(t, err)
	assert.Contains(t, string(plist), "<string>/Users/me/R&amp;D &lt;tools&gt;/pomo</string>")
}

func TestLoginItemRejectsTemporaryBinary(t *testing.T) {
	login := testLoginItem(t)
	login.Executable = "/var/folders/xy/T/go-build123/b001/exe/pomo"

	assert.Error(t, login.Enable())
	assert.False(t, login.Enabled())
}
