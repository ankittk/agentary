package cli

import (
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/daemon"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running Agentary daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			stopped, err := daemon.Stop(cmd.Context(), home)
			if err != nil {
				return err
			}
			if !stopped {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Agentary is not running")
				return nil
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Stopped")
			return nil
		},
	}
	return cmd
}
