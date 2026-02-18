//go:build linux || darwin

package daemon

import (
	"os"
	"os/exec"
	"syscall"
)

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func processExists(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func signalTerm(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
