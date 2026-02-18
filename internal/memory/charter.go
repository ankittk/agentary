package memory

import (
	"os"
)

// ReadCharter returns the contents of the team's charter.md. Returns empty string if missing.
func ReadCharter(teamDir string) (string, error) {
	path := CharterPath(teamDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// WriteCharter writes the team charter. Creates teamDir if needed.
func WriteCharter(teamDir, content string) error {
	if err := os.MkdirAll(teamDir, 0o755); err != nil {
		return err
	}
	path := CharterPath(teamDir)
	return os.WriteFile(path, []byte(content), 0o644)
}

// CharterPath is in paths.go
