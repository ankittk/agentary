package cli

import (
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/daemon"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Agentary daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			st, err := daemon.Status(cmd.Context(), home)
			if err != nil {
				return err
			}
			if !st.Running {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Agentary not running")
				return nil
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Agentary running (pid %d, addr %s)\n", st.PID, st.Addr)
			return nil
		},
	}
	return cmd
}
