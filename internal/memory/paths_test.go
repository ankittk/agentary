package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeTeamName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"team1", "team1"},
		{"  team 2  ", "team_2"},
		{"a b c", "a_b_c"},
	}
	for _, tt := range tests {
		if got := SafeTeamName(tt.in); got != tt.want {
			t.Errorf("SafeTeamName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSafeAgentName(t *testing.T) {
	t.Parallel()
	if got := SafeAgentName("alice"); got != "alice" {
		t.Errorf("SafeAgentName(alice) = %q", got)
	}
	if got := SafeAgentName("  bob  "); got != "bob" {
		t.Errorf("SafeAgentName('  bob  ') = %q", got)
	}
}

func TestTeamDir(t *testing.T) {
	t.Parallel()
	got := TeamDir("/home", "team one")
	want := filepath.Join("/home", "teams", "team_one")
	if got != want {
		t.Errorf("TeamDir: got %q, want %q", got, want)
	}
}

func TestAgentDir(t *testing.T) {
	t.Parallel()
	teamDir := filepath.Join("/home", "teams", "t1")
	got := AgentDir(teamDir, "agent one")
	want := filepath.Join(teamDir, "agents", "agent_one")
	if got != want {
		t.Errorf("AgentDir: got %q, want %q", got, want)
	}
}

func TestSharedDir_CharterPath_JournalPath_AgentConfigPath(t *testing.T) {
	t.Parallel()
	teamDir := filepath.Join(os.TempDir(), "agentary-test", "teams", "t1")
	if got := SharedDir(teamDir); got != filepath.Join(teamDir, "shared") {
		t.Errorf("SharedDir: got %q", got)
	}
	if got := CharterPath(teamDir); got != filepath.Join(teamDir, "charter.md") {
		t.Errorf("CharterPath: got %q", got)
	}
	agentDir := filepath.Join(teamDir, "agents", "a1")
	if got := JournalPath(agentDir); got != filepath.Join(agentDir, "journal.md") {
		t.Errorf("JournalPath: got %q", got)
	}
	if got := AgentConfigPath(agentDir); got != filepath.Join(agentDir, "config.yaml") {
		t.Errorf("AgentConfigPath: got %q", got)
	}
	if got := NotesDir(agentDir); got != filepath.Join(agentDir, "notes") {
		t.Errorf("NotesDir: got %q", got)
	}
}
