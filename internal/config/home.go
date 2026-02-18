package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

type homeKey struct{}

// WithHome stores the agentary home path in the context.
func WithHome(ctx context.Context, home string) context.Context {
	return context.WithValue(ctx, homeKey{}, home)
}

// HomeFrom returns the agentary home path from the context, if set.
func HomeFrom(ctx context.Context) (string, bool) {
	v := ctx.Value(homeKey{})
	s, ok := v.(string)
	return s, ok
}

// MustHomeFrom returns the home path from the context, or panics if not set.
func MustHomeFrom(ctx context.Context) string {
	if h, ok := HomeFrom(ctx); ok && h != "" {
		return h
	}
	panic("agentary home missing from context")
}

// ResolveHome returns the agentary home directory (override, AGENTARY_HOME, or default ~/.agentary).
func ResolveHome(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	if env := os.Getenv("AGENTARY_HOME"); env != "" {
		return filepath.Clean(env), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("could not determine user home directory")
	}
	return filepath.Join(home, ".agentary"), nil
}
