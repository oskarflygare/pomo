//go:build !darwin || !cgo

package menubar

import (
	"errors"

	"github.com/Bahaaio/pomo/config"
)

// Run reports that the menu bar is unavailable on this build.
func Run(config.Config) error {
	return errors.New("the menu bar timer requires macOS and a build with cgo enabled")
}
