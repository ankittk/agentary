//go:build windows

package daemon

import (
	"os"
	"os/exec"
)

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	// No Setsid on Windows; process runs in same console by default.
}

func processExists(pid int) bool {
	// On Windows there is no kill(pid, 0). We could use OpenProcess from golang.org/x/sys/windows;
	// for now assume process exists if pid is valid (caller may get connection refused if daemon died).
	return pid > 0
}

func signalTerm(proc *os.Process) error {
	// On Windows, SIGTERM is not supported; use Kill to terminate.
	return proc.Kill()
}
