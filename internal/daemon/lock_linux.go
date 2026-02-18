//go:build !windows

package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type daemonLock struct {
	f *os.File
}

func acquireLock(lockFile string) (*daemonLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockFile), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("agentary is already running (could not acquire lock)")
		}
		return nil, err
	}
	return &daemonLock{f: f}, nil
}

func (l *daemonLock) release() {
	if l == nil || l.f == nil {
		return
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	_ = l.f.Close()
}
