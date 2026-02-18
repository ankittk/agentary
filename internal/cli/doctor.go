package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/ankittk/agentary/internal/config"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify runtime dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = config.MustHomeFrom(cmd.Context()) // currently unused, but ensures home resolves

			var problems []string

			// git is required for interacting with local repos.
			if _, err := exec.LookPath("git"); err != nil {
				problems = append(problems, "missing dependency: git (not found on PATH)")
			}

			if len(problems) > 0 {
				for _, p := range problems {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), p)
				}
				return errors.New("doctor checks failed")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
	return cmd
}
