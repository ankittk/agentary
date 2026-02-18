package cli

import (
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/identity"
	"github.com/spf13/cobra"
)

func newIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity",
		Short: "Manage human identity (for commit attribution and review)",
	}
	cmd.AddCommand(newIdentityDetectCmd())
	return cmd
}

func newIdentityDetectCmd() *cobra.Command {
	var repoDir string
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect identity from git config and save to members/",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			h, err := identity.DetectAndSave(home, repoDir)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Detected: %s <%s>\n", h.Name, h.Email)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved to %s\n", identity.MemberPath(home, h.Name))
			return nil
		},
	}
	cmd.Flags().StringVar(&repoDir, "repo", "", "Git repo path (default: global git config)")
	return cmd
}
