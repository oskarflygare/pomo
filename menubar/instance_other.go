//go:build !unix

package menubar

import "fmt"

// InstanceLock is a no-op where file locking is not implemented; the menu bar
// timer itself only runs on macOS.
type InstanceLock struct{}

// AlreadyRunningError reports that another process holds the instance lock.
type AlreadyRunningError struct {
	PID int
}

func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("menu bar timer is already running (pid %d)", e.PID)
}

func AcquireInstanceLock(string) (*InstanceLock, error) { return &InstanceLock{}, nil }

func (l *InstanceLock) Release() error { return nil }
