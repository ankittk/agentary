package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWithHome_HomeFrom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if _, ok := HomeFrom(ctx); ok {
		t.Fatal("expected no home in empty context")
	}
	ctx = WithHome(ctx, "/foo/bar")
	got, ok := HomeFrom(ctx)
	if !ok || got != "/foo/bar" {
		t.Fatalf("HomeFrom: got %q, ok=%v; want /foo/bar, true", got, ok)
	}
}

func TestMustHomeFrom(t *testing.T) {
	t.Parallel()
	ctx := WithHome(context.Background(), "/agentary")
	if got := MustHomeFrom(ctx); got != "/agentary" {
		t.Fatalf("MustHomeFrom: got %q", got)
	}
}

func TestMustHomeFrom_panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when home missing")
		}
	}()
	MustHomeFrom(context.Background())
}

func TestResolveHome_override(t *testing.T) {
	t.Parallel()
	got, err := ResolveHome("/custom/home")
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	if got != filepath.Clean("/custom/home") {
		t.Fatalf("ResolveHome: got %q", got)
	}
}

func TestResolveHome_env(t *testing.T) {
	t.Setenv("AGENTARY_HOME", "/env/home")
	got, err := ResolveHome("")
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	if got != filepath.Clean("/env/home") {
		t.Fatalf("ResolveHome from env: got %q", got)
	}
}

func TestResolveHome_default(t *testing.T) {
	t.Setenv("AGENTARY_HOME", "")
	// Override empty so we use UserHomeDir
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("UserHomeDir: %v", err)
	}
	got, err := ResolveHome("")
	if err != nil {
		t.Fatalf("ResolveHome: %v", err)
	}
	want := filepath.Join(home, ".agentary")
	if got != want {
		t.Fatalf("ResolveHome default: got %q, want %q", got, want)
	}
}
