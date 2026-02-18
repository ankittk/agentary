package memory

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// JournalEntry represents one entry appended to an agent's journal.
type JournalEntry struct {
	TaskID    int64
	TaskTitle string
	Outcome   string
	Decisions string
	Patterns  string
	CreatedAt time.Time
}

// Journal manages an agent's journal.md file: append entries and read/summarize.
type Journal struct {
	AgentName string
	TeamDir   string
}

// Append adds an entry to the agent's journal. Creates the agent directory and
// journal file if they do not exist. The entry is appended in markdown form.
func (j *Journal) Append(ctx context.Context, entry JournalEntry) error {
	agentDir := AgentDir(j.TeamDir, j.AgentName)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}
	path := JournalPath(agentDir)
	block := formatJournalBlock(entry)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(block); err != nil {
		return fmt.Errorf("write journal: %w", err)
	}
	return nil
}

func formatJournalBlock(e JournalEntry) string {
	var b strings.Builder
	b.WriteString("\n---\n\n")
	b.WriteString("## ")
	b.WriteString(e.CreatedAt.Format("2006-01-02 15:04"))
	if e.TaskTitle != "" {
		b.WriteString(" — ")
		b.WriteString(e.TaskTitle)
	}
	b.WriteString("\n\n")
	if e.TaskID > 0 {
		b.WriteString("- **Task:** ")
		b.WriteString(fmt.Sprintf("%d", e.TaskID))
		b.WriteString("\n")
	}
	if e.Outcome != "" {
		b.WriteString("- **Outcome:** ")
		b.WriteString(e.Outcome)
		b.WriteString("\n")
	}
	if e.Decisions != "" {
		b.WriteString("- **Decisions:** ")
		b.WriteString(e.Decisions)
		b.WriteString("\n")
	}
	if e.Patterns != "" {
		b.WriteString("- **Patterns:** ")
		b.WriteString(e.Patterns)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// Read returns up to limit journal entries from the end of the file (most recent first
// is not implemented here; we read the file and return the last N blocks). For simplicity,
// Read returns the raw markdown tail of the file (last approximate size). A limit of 0
// means return the whole file content.
func (j *Journal) Read(ctx context.Context, limitBytes int) (string, error) {
	path := JournalPath(AgentDir(j.TeamDir, j.AgentName))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	s := string(data)
	if limitBytes <= 0 || len(s) <= limitBytes {
		return s, nil
	}
	return s[len(s)-limitBytes:], nil
}

// Summary returns a short summary of the journal for context (e.g. last N characters
// or a placeholder). Useful for injecting into agent context.
func (j *Journal) Summary(ctx context.Context, maxLen int) (string, error) {
	if maxLen <= 0 {
		maxLen = 4000
	}
	s, err := j.Read(ctx, maxLen)
	if err != nil {
		return "", err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no journal entries yet)", nil
	}
	return s, nil
}

// EnsureAgentDir creates the agent directory and notes subdirectory if they do not exist.
func EnsureAgentDir(teamDir, agentName string) error {
	agentDir := AgentDir(teamDir, agentName)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}
	notesPath := NotesDir(agentDir)
	return os.MkdirAll(notesPath, 0o755)
}

// EnsureTeamDirs creates the team directory and shared/ subdirectory if they do not exist.
func EnsureTeamDirs(teamDir string) error {
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(SharedDir(teamDir), 0o755)
}
