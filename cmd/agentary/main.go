package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// Version is set at build time via -ldflags "-X main.Version=..."
var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	os.Exit(Run(ctx, os.Args[1:]))
}
