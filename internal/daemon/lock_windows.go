//go:build windows

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

type daemonLock struct {
	f    *os.File
	path string
}

func acquireLock(lockFile string) (*daemonLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockFile), 0o755); err != nil {
		return nil, err
	}
	// On Windows, open with exclusive create: only one process can have the file.
	// If it already exists, we get "already running".
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("agentary is already running (could not acquire lock)")
		}
		return nil, err
	}
	return &daemonLock{f: f, path: lockFile}, nil
}

func (l *daemonLock) release() {
	if l == nil || l.f == nil {
		return
	}
	_ = l.f.Close()
	_ = os.Remove(l.path)
}
