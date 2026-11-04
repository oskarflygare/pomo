package menubar

import (
	"path/filepath"

	"github.com/Bahaaio/pomo/db"
)

// DefaultLockPath is the instance lock file, kept next to the database.
func DefaultLockPath() (string, error) {
	dir, err := db.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "menubar.lock"), nil
}
