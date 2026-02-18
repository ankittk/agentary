package memory

import (
	"path/filepath"
	"strings"
)

// SafeTeamName returns a filesystem-safe version of the team name (e.g. for directory names).
func SafeTeamName(teamName string) string {
	return strings.ReplaceAll(strings.TrimSpace(teamName), " ", "_")
}

// SafeAgentName returns a filesystem-safe version of the agent name.
func SafeAgentName(agentName string) string {
	return strings.ReplaceAll(strings.TrimSpace(agentName), " ", "_")
}

// TeamDir returns the path to a team's directory under home: <home>/teams/<safe_team_name>/.
func TeamDir(home, teamName string) string {
	return filepath.Join(home, "teams", SafeTeamName(teamName))
}

// AgentDir returns the path to an agent's directory: <teamDir>/agents/<safe_agent_name>/.
func AgentDir(teamDir, agentName string) string {
	return filepath.Join(teamDir, "agents", SafeAgentName(agentName))
}

// SharedDir returns the team shared directory: <teamDir>/shared/.
func SharedDir(teamDir string) string {
	return filepath.Join(teamDir, "shared")
}

// CharterPath returns the path to the team charter: <teamDir>/charter.md.
func CharterPath(teamDir string) string {
	return filepath.Join(teamDir, "charter.md")
}

// JournalPath returns the path to an agent's journal: <agentDir>/journal.md.
func JournalPath(agentDir string) string {
	return filepath.Join(agentDir, "journal.md")
}

// NotesDir returns the path to an agent's notes directory: <agentDir>/notes/.
func NotesDir(agentDir string) string {
	return filepath.Join(agentDir, "notes")
}

// AgentConfigPath returns the path to an agent's config: <agentDir>/config.yaml.
func AgentConfigPath(agentDir string) string {
	return filepath.Join(agentDir, "config.yaml")
}
