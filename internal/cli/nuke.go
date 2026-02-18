package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ankittk/agentary/internal/config"
	"github.com/spf13/cobra"
)

func newNukeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nuke",
		Short: "Destroy all Agentary state under AGENTARY_HOME",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "WARNING: this will permanently delete all Agentary data.")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Directory: %s\n", home)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), `Type "delete everything" to confirm:`)

			in := bufio.NewReader(cmd.InOrStdin())
			line, err := in.ReadString('\n')
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				return err
			}
			line = strings.TrimSpace(line)
			if line != "delete everything" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}

			if err := os.RemoveAll(home); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Deleted.")
			return nil
		},
	}
	return cmd
}
