package identity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Human holds a human identity (name, email) for commit attribution and review approvals.
type Human struct {
	Name   string `yaml:"name"`
	Email  string `yaml:"email"`
	Source string `yaml:"source,omitempty"` // e.g. "git"
}

// DetectFromGit runs `git config user.name` and `git config user.email` (in repoDir, or global if repoDir is empty)
// and returns a Human. If either command fails, returns empty name/email for that field.
func DetectFromGit(repoDir string) (Human, error) {
	var h Human
	h.Source = "git"
	name, err := gitConfig(repoDir, "user.name")
	if err == nil {
		h.Name = strings.TrimSpace(name)
	}
	email, err := gitConfig(repoDir, "user.email")
	if err == nil {
		h.Email = strings.TrimSpace(email)
	}
	return h, nil
}

func gitConfig(repoDir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// MembersDir returns the path to the members directory: <home>/members/.
func MembersDir(home string) string {
	return filepath.Join(home, "members")
}

// MemberPath returns the path to a member file: <home>/members/<username>.yaml.
// Username is sanitized for filesystem (spaces -> _, lowercase for consistency).
func MemberPath(home, username string) string {
	safe := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(username), " ", "_"))
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(MembersDir(home), safe+".yaml")
}

// LoadHuman loads a human identity from <home>/members/<username>.yaml.
func LoadHuman(home, username string) (*Human, error) {
	path := MemberPath(home, username)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var h Human
	if err := yaml.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// SaveHuman writes the human identity to <home>/members/<username>.yaml.
func SaveHuman(home, username string, h *Human) error {
	dir := MembersDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}
	return os.WriteFile(MemberPath(home, username), data, 0o644)
}

// DetectAndSave runs DetectFromGit and saves to members/<username>.yaml.
// Username is derived from h.Name or h.Email (part before @) after detection.
func DetectAndSave(home, repoDir string) (*Human, error) {
	h, err := DetectFromGit(repoDir)
	if err != nil {
		return nil, err
	}
	username := h.Name
	if username == "" {
		if idx := strings.Index(h.Email, "@"); idx > 0 {
			username = h.Email[:idx]
		}
	}
	if username == "" {
		username = "default"
	}
	if err := SaveHuman(home, username, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
