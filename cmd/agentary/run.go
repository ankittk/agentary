package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ankittk/agentary/internal/cli"
)

func Run(ctx context.Context, args []string) int {
	root := cli.NewRootCmd(Version)
	root.SetArgs(args)
	if err := root.ExecuteContext(ctx); err != nil {
		// cobra already prints usage for some error types; keep this minimal.
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
