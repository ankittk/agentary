package sandbox

import (
	"path/filepath"
	"strings"
)

// WriteGuard enforces write-path isolation per role. Each tool call that
// writes to the filesystem should be checked with AllowWrite(path) before
// execution. Manager can write anywhere under the team directory; engineer
// can only write to their agent dir, task worktrees, and team shared/.
type WriteGuard struct {
	Role         string   // "manager" or "engineer"
	AgentName    string
	TeamDir      string   // e.g. ~/.agentary/teams/<team>/
	WorktreeDirs []string // task worktree paths (engineer only)
}

// AllowWrite returns true if the guard allows writing to the given path.
// Paths are normalized (cleaned and absolutized when possible). Manager may
// write anywhere under TeamDir. Engineer may write only to:
//   - TeamDir/agents/<AgentName>/ (own agent dir)
//   - Any path under an entry in WorktreeDirs (task worktrees)
//   - TeamDir/shared/ (team shared folder)
func (g *WriteGuard) AllowWrite(path string) bool {
	if path == "" {
		return false
	}
	clean := filepath.Clean(path)
	abs, err := filepath.Abs(clean)
	if err != nil {
		abs = clean
	}
	teamDir := g.normalizeDir(g.TeamDir)
	if teamDir != "" && abs != teamDir && !strings.HasPrefix(abs, teamDir+string(filepath.Separator)) {
		// Path must be under team dir for both roles
		return false
	}
	if g.Role == "manager" {
		return true
	}
	// Engineer: only agent dir, worktrees, or shared/
	agentDir := filepath.Join(teamDir, "agents", g.AgentName)
	if agentDir != "" && (abs == agentDir || strings.HasPrefix(abs, agentDir+string(filepath.Separator))) {
		return true
	}
	sharedDir := filepath.Join(teamDir, "shared")
	if sharedDir != "" && (abs == sharedDir || strings.HasPrefix(abs, sharedDir+string(filepath.Separator))) {
		return true
	}
	for _, wd := range g.WorktreeDirs {
		d := g.normalizeDir(wd)
		if d != "" && (abs == d || strings.HasPrefix(abs, d+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

func (g *WriteGuard) normalizeDir(dir string) string {
	if dir == "" {
		return ""
	}
	clean := filepath.Clean(dir)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean
	}
	return abs
}
