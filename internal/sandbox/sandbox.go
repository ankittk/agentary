package sandbox

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
)

// WrapCommand returns an *exec.Cmd that runs binary with args. If home is non-empty and
// bubblewrap (bwrap) is available on Linux, the command runs inside a minimal bubblewrap
// sandbox. If teamDir is non-empty, only teamDir is writable and home is read-only (so
// protected/ under home cannot be written). Otherwise the whole home is writable.
// Use teamDir when running an agent so it can only write under the team directory.
func WrapCommand(ctx context.Context, home, teamDir, binary string, args []string) *exec.Cmd {
	if home == "" || runtime.GOOS != "linux" {
		return exec.CommandContext(ctx, binary, args...)
	}
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return exec.CommandContext(ctx, binary, args...)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return exec.CommandContext(ctx, binary, args...)
	}
	var bwrapArgs []string
	if teamDir != "" {
		absTeam, _ := filepath.Abs(teamDir)
		if absTeam != "" && (absTeam == absHome || (len(absTeam) > len(absHome) && absTeam[len(absHome)] == filepath.Separator)) {
			// Restrict writes to team dir only: home (and thus protected/) read-only, team dir rw.
			bwrapArgs = []string{
				"--ro-bind", absHome, absHome,
				"--bind", absTeam, absTeam,
				"--ro-bind", "/usr", "/usr",
				"--ro-bind", "/lib", "/lib",
				"--ro-bind", "/lib64", "/lib64",
				"--dev", "/dev",
				"--proc", "/proc",
				"--tmpfs", "/tmp",
				"--unshare-pid",
			}
		}
	}
	if bwrapArgs == nil {
		bwrapArgs = []string{
			"--bind", absHome, absHome,
			"--ro-bind", "/usr", "/usr",
			"--ro-bind", "/lib", "/lib",
			"--ro-bind", "/lib64", "/lib64",
			"--dev", "/dev",
			"--proc", "/proc",
			"--tmpfs", "/tmp",
			"--unshare-pid",
		}
	}
	bwrapArgs = append(bwrapArgs, "--", binary)
	bwrapArgs = append(bwrapArgs, args...)
	return exec.CommandContext(ctx, bwrap, bwrapArgs...)
}
