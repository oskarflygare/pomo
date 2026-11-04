//go:build unix

package menubar

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// InstanceLock is an exclusive lock on a file, held for as long as one menu
// bar timer runs. The kernel releases it when the process exits, so a crash
// never leaves a stale lock behind.
type InstanceLock struct {
	file *os.File
}

// AlreadyRunningError reports that another process holds the instance lock.
type AlreadyRunningError struct {
	PID int // 0 when unknown
}

func (e *AlreadyRunningError) Error() string {
	if e.PID > 0 {
		return fmt.Sprintf("menu bar timer is already running (pid %d)", e.PID)
	}
	return "menu bar timer is already running"
}

// AcquireInstanceLock takes the lock at path without blocking and records the
// current pid in it. It returns *AlreadyRunningError if another process holds it.
func AcquireInstanceLock(path string) (*InstanceLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		defer file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, &AlreadyRunningError{PID: readPID(file)}
		}
		return nil, err
	}

	if err := file.Truncate(0); err == nil {
		_, _ = file.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	}

	return &InstanceLock{file: file}, nil
}

// Release gives up the lock. The file is left in place for the next instance.
func (l *InstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := l.file.Close() // closing the descriptor drops the flock
	l.file = nil
	return err
}

func readPID(file *os.File) int {
	buf := make([]byte, 32)
	n, _ := file.ReadAt(buf, 0)
	pid, err := strconv.Atoi(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		return 0
	}
	return pid
}
