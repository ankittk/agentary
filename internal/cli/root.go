package cli

import (
	"os"

	"github.com/ankittk/agentary/internal/config"
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	var homeOverride string

	cmd := &cobra.Command{
		Use:          "agentary",
		Short:        "Agentary — local agent orchestration with a web UI",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			home, err := config.ResolveHome(homeOverride)
			if err != nil {
				return err
			}
			cmd.SetContext(config.WithHome(cmd.Context(), home))
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&homeOverride, "home", "", "Override Agentary home directory (default: ~/.agentary, env: AGENTARY_HOME)")

	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())

	cmd.AddCommand(newTeamCmd())
	cmd.AddCommand(newTaskCmd())
	cmd.AddCommand(newAgentCmd())
	cmd.AddCommand(newRepoCmd())
	cmd.AddCommand(newWorkflowCmd())
	cmd.AddCommand(newNetworkCmd())
	cmd.AddCommand(newIdentityCmd())
	cmd.AddCommand(newApikeyCmd())
	cmd.AddCommand(newNukeCmd())

	// Hidden internal subcommand used by `agentary start` for background mode.
	cmd.AddCommand(newDaemonCmd())

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	cmd.SetVersionTemplate("{{.Version}}\n")
	if version != "" {
		cmd.Version = version
	} else {
		cmd.Version = "dev"
	}

	return cmd
}
