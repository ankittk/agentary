package postgres

import (
	"context"
	"os"
	"testing"
)

func TestOpen_skipIfNoDatabaseURL(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping postgres test")
	}
	st, err := Open(dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	teams, err := st.ListTeams(ctx)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if teams == nil {
		t.Fatal("teams should not be nil")
	}
}
