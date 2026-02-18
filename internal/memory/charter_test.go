package memory

import (
	"path/filepath"
	"testing"
)

func TestReadCharter_WriteCharter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "t1")

	// Missing file returns empty, nil
	content, err := ReadCharter(teamDir)
	if err != nil {
		t.Fatalf("ReadCharter missing: %v", err)
	}
	if content != "" {
		t.Fatalf("ReadCharter missing: got %q", content)
	}

	// Write then read
	if err := WriteCharter(teamDir, "# Team charter"); err != nil {
		t.Fatalf("WriteCharter: %v", err)
	}
	content, err = ReadCharter(teamDir)
	if err != nil {
		t.Fatalf("ReadCharter: %v", err)
	}
	if content != "# Team charter" {
		t.Fatalf("ReadCharter: got %q", content)
	}
}
