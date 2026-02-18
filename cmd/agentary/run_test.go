package main

import (
	"context"
	"testing"
)

func TestRun_help(t *testing.T) {
	ctx := context.Background()
	code := Run(ctx, []string{"--help"})
	if code != 0 {
		t.Errorf("Run --help: got exit code %d", code)
	}
}

func TestRun_version(t *testing.T) {
	ctx := context.Background()
	code := Run(ctx, []string{"--version"})
	if code != 0 {
		t.Errorf("Run --version: got exit code %d", code)
	}
}

func TestRun_unknownFlag(t *testing.T) {
	ctx := context.Background()
	code := Run(ctx, []string{"--unknown-flag"})
	if code != 1 {
		t.Errorf("Run --unknown-flag: got exit code %d, want 1", code)
	}
}

