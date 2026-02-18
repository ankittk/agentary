package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/memory"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams",
	}
	cmd.AddCommand(newTeamAddCmd())
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamRemoveCmd())
	return cmd
}

func newTeamAddCmd() *cobra.Command {
	var name string
	var agents int
	var repo string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a team (optionally with --agents N and --repo path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("--name is required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			t, err := st.CreateTeam(cmd.Context(), name)
			if err != nil {
				return err
			}
			_ = memory.EnsureTeamDirs(memory.TeamDir(home, t.Name))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created team %q (%s)\n", t.Name, t.TeamID)

			teamDir := memory.TeamDir(home, name)
			for i := 0; i < agents; i++ {
				agentName := fmt.Sprintf("agent-%d", i+1)
				if err := st.CreateAgent(cmd.Context(), name, agentName, "engineer"); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not add agent %q: %v\n", agentName, err)
				} else {
					_ = memory.EnsureAgentDir(teamDir, agentName)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added agent %q\n", agentName)
				}
			}
			if repo != "" {
				repoName := "default"
				if err := st.CreateRepo(cmd.Context(), name, repoName, repo, "manual", nil); err != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not add repo: %v\n", err)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added repo %q (source %q)\n", repoName, repo)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().IntVar(&agents, "agents", 0, "Create N agents (agent-1, agent-2, ...)")
	cmd.Flags().StringVar(&repo, "repo", "", "Register a repo for the team (source path or URL)")
	return cmd
}

func newTeamListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			teams, err := st.ListTeams(cmd.Context())
			if err != nil {
				return err
			}
			if len(teams) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No teams.")
				return nil
			}
			for _, t := range teams {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s (agents=%d tasks=%d)\n", t.Name, t.AgentCount, t.TaskCount)
			}
			return nil
		},
	}
	return cmd
}

func newTeamRemoveCmd() *cobra.Command {
	var (
		name string
		yes  bool
	)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a team and its data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("--name is required")
			}
			if !yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Remove team %q and all its data? Type the team name to confirm:\n", name)
				in := bufio.NewReader(cmd.InOrStdin())
				line, err := in.ReadString('\n')
				if err != nil && !strings.Contains(err.Error(), "EOF") {
					return err
				}
				if strings.TrimSpace(line) != name {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.DeleteTeam(cmd.Context(), name); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Removed.")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
