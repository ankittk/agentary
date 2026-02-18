package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenClose(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// Second Migrate is idempotent
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate again: %v", err)
	}
}
