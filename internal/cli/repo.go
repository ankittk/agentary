package cli

import (
	"errors"
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
	}
	cmd.AddCommand(newRepoAddCmd())
	cmd.AddCommand(newRepoListCmd())
	cmd.AddCommand(newRepoSetApprovalCmd())
	return cmd
}

func newRepoAddCmd() *cobra.Command {
	var (
		team     string
		name     string
		source   string
		approval string
		testCmd  string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register a repo for a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return errors.New("--team is required")
			}
			if name == "" {
				return errors.New("--name is required")
			}
			if source == "" {
				return errors.New("--source is required")
			}
			if approval != "auto" && approval != "manual" {
				return errors.New("--approval must be auto or manual")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			var tc *string
			if testCmd != "" {
				tc = &testCmd
			}
			if err := st.CreateRepo(cmd.Context(), team, name, source, approval, tc); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Registered repo %q for %q\n", name, team)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().StringVar(&name, "name", "", "Repo name")
	cmd.Flags().StringVar(&source, "source", "", "Repo source path or URL")
	cmd.Flags().StringVar(&approval, "approval", "manual", "Approval mode: auto or manual")
	cmd.Flags().StringVar(&testCmd, "test-cmd", "", "Optional test command")
	return cmd
}

func newRepoListCmd() *cobra.Command {
	var team string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repos for a team",
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

			repos, err := st.ListRepos(cmd.Context(), team)
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No repos.")
				return nil
			}
			for _, r := range repos {
				line := fmt.Sprintf("- %s (%s) approval=%s", r.Name, r.Source, r.Approval)
				if r.TestCmd != nil {
					line += " test_cmd=" + *r.TestCmd
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	return cmd
}

func newRepoSetApprovalCmd() *cobra.Command {
	var team, name, approval string
	cmd := &cobra.Command{
		Use:   "set-approval",
		Short: "Set repo approval mode (auto or manual)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || name == "" {
				return errors.New("--team and --name are required")
			}
			if approval != "auto" && approval != "manual" {
				return errors.New("--approval must be auto or manual")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()
			if err := st.SetRepoApproval(cmd.Context(), team, name, approval); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo %q approval set to %q\n", name, approval)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().StringVar(&name, "name", "", "Repo name")
	cmd.Flags().StringVar(&approval, "approval", "manual", "Approval mode: auto or manual")
	return cmd
}
