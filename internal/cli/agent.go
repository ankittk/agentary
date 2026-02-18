package cli

import (
	"errors"
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/memory"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
	}
	cmd.AddCommand(newAgentAddCmd())
	cmd.AddCommand(newAgentListCmd())
	return cmd
}

func newAgentAddCmd() *cobra.Command {
	var (
		team string
		name string
		role string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an agent to a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return errors.New("--team is required")
			}
			if name == "" {
				return errors.New("--name is required")
			}

			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.CreateAgent(cmd.Context(), team, name, role); err != nil {
				return err
			}
			_ = memory.EnsureAgentDir(memory.TeamDir(home, team), name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added agent %q to %q (role=%s)\n", name, team, roleOrDefault(role))
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&role, "role", "engineer", "Agent role")
	return cmd
}

func newAgentListCmd() *cobra.Command {
	var team string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agents on a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return errors.New("--team is required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			agents, err := st.ListAgents(cmd.Context(), team)
			if err != nil {
				return err
			}
			if len(agents) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No agents.")
				return nil
			}
			for _, a := range agents {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", a.Name, a.Role)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	return cmd
}

func roleOrDefault(role string) string {
	if role == "" {
		return "engineer"
	}
	return role
}
