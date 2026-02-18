package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMembersDir(t *testing.T) {
	t.Parallel()
	got := MembersDir("/home")
	if got != filepath.Join("/home", "members") {
		t.Fatalf("MembersDir: got %q", got)
	}
}

func TestMemberPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		home     string
		username string
		wantSuffix string
	}{
		{"/home", "alice", "alice.yaml"},
		{"/home", "Alice Bob", "alice_bob.yaml"},
		{"/home", "  default  ", "default.yaml"},
		{"/home", "", "default.yaml"},
	}
	for _, tt := range tests {
		got := MemberPath(tt.home, tt.username)
		if filepath.Base(got) != tt.wantSuffix {
			t.Errorf("MemberPath(%q, %q) base = %q, want %q", tt.home, tt.username, filepath.Base(got), tt.wantSuffix)
		}
	}
}

func TestSaveHuman_LoadHuman(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := &Human{Name: "Test", Email: "test@example.com", Source: "git"}
	if err := SaveHuman(dir, "testuser", h); err != nil {
		t.Fatalf("SaveHuman: %v", err)
	}
	loaded, err := LoadHuman(dir, "testuser")
	if err != nil {
		t.Fatalf("LoadHuman: %v", err)
	}
	if loaded == nil || loaded.Name != "Test" || loaded.Email != "test@example.com" {
		t.Fatalf("LoadHuman: got %+v", loaded)
	}
}

func TestLoadHuman_missingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	loaded, err := LoadHuman(dir, "nonexistent")
	if err != nil {
		t.Fatalf("LoadHuman: %v", err)
	}
	if loaded != nil {
		t.Fatalf("LoadHuman missing file: expected nil, got %+v", loaded)
	}
}

func TestLoadHuman_invalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	membersDir := filepath.Join(dir, "members")
	if err := os.MkdirAll(membersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(membersDir, "bad.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadHuman(dir, "bad")
	if err == nil {
		t.Fatal("LoadHuman: expected error for invalid YAML")
	}
}
