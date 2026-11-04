//go:build unix

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// detach re-runs the current binary with args in a new session, with no
// controlling terminal, so it survives the terminal that started it closing.
func detach(args ...string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()

	child := exec.Command(exe, args...)
	child.Stdin, child.Stdout, child.Stderr = devNull, devNull, devNull
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := child.Start(); err != nil {
		return 0, err
	}
	pid := child.Process.Pid
	return pid, child.Process.Release()
}
