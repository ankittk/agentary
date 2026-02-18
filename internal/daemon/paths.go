package daemon

import (
	"path/filepath"
)

func protectedDir(home string) string {
	return filepath.Join(home, "protected")
}

func pidPath(home string) string {
	return filepath.Join(protectedDir(home), "daemon.pid")
}

func lockPath(home string) string {
	return filepath.Join(protectedDir(home), "daemon.lock")
}

func addrPath(home string) string {
	return filepath.Join(protectedDir(home), "daemon.addr")
}
