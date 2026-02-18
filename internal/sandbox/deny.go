package sandbox

import (
	"strings"
)

// bashDenyList contains substrings that must not appear in shell command lines
// before execution. Used to block dangerous or destructive commands.
var bashDenyList = []string{
	"sqlite3",
	"DROP TABLE",
	"DELETE FROM",
	"rm -rf .git",
	"rm -rf .git/",
	"chmod 777",
	"curl | sh",
	"wget | sh",
	"curl | bash",
	"wget | bash",
	"| sh",
	"| bash",
	"eval $(",
	"> /dev/sd",
	"mkfs.",
	":(){ :|:& };:", // fork bomb
}

// disallowedGitCommands are git command prefixes that agents must not run.
// Branch topology (rebase, merge, push, etc.) is managed by the daemon only.
var disallowedGitCommands = []string{
	"git rebase",
	"git merge",
	"git pull",
	"git push",
	"git fetch",
	"git checkout",
	"git switch",
	"git reset --hard",
	"git worktree",
	"git branch ",
	"git branch -",
	"git remote",
	"git filter-branch",
	"git reflog expire",
}

// BlockedShellCommand returns true if the command line (typically a single
// shell command or script snippet) contains any denied substring. Matching
// is case-insensitive. Call this before executing shell commands from agent output.
func BlockedShellCommand(cmdLine string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmdLine))
	for _, deny := range bashDenyList {
		if strings.Contains(lower, strings.ToLower(deny)) {
			return true
		}
	}
	return false
}

// BlockedGitCommand returns true if the given git arguments (e.g. after
// "git" in argv) represent a disallowed git command. Pass the full args
// slice; the first element is typically the subcommand (e.g. "rebase").
// Used to prevent agents from running topology-changing or dangerous git ops.
func BlockedGitCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	// Rebuild a single string to match disallowedGitCommands (e.g. "git rebase")
	cmdLine := "git " + strings.TrimSpace(strings.Join(args, " "))
	lower := strings.ToLower(cmdLine)
	for _, dis := range disallowedGitCommands {
		if strings.HasPrefix(lower, strings.ToLower(dis)) {
			return true
		}
	}
	return false
}
