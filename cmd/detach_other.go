//go:build !unix

package cmd

import "errors"

func detach(args ...string) (int, error) {
	return 0, errors.New("running in the background is not supported on this platform")
}
