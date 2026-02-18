package cli

import (
	"errors"
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage workflows",
	}
	cmd.AddCommand(newWorkflowInitCmd())
	cmd.AddCommand(newWorkflowAddCmd())
	cmd.AddCommand(newWorkflowListCmd())
	cmd.AddCommand(newWorkflowShowCmd())
	return cmd
}

func newWorkflowInitCmd() *cobra.Command {
	var team string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize default workflow metadata for a team",
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

			// Best-effort idempotent insert.
			_, _ = st.CreateWorkflow(cmd.Context(), team, "default", 1, "builtin:default")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Initialized workflow default v1 for %q\n", team)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	return cmd
}

func newWorkflowAddCmd() *cobra.Command {
	var (
		team       string
		name       string
		version    int
		sourcePath string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register workflow metadata for a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return errors.New("--team is required")
			}
			if name == "" {
				return errors.New("--name is required")
			}
			if version <= 0 {
				return errors.New("--version must be > 0")
			}
			if sourcePath == "" {
				return errors.New("--source is required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if _, err := st.CreateWorkflow(cmd.Context(), team, name, version, sourcePath); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Registered workflow %s v%d for %q\n", name, version, team)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().StringVar(&name, "name", "", "Workflow name")
	cmd.Flags().IntVar(&version, "version", 1, "Workflow version (>0)")
	cmd.Flags().StringVar(&sourcePath, "source", "", "Source path (file path or builtin:...)")
	return cmd
}

func newWorkflowListCmd() *cobra.Command {
	var team string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflows for a team",
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

			wfs, err := st.ListWorkflows(cmd.Context(), team)
			if err != nil {
				return err
			}
			if len(wfs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No workflows.")
				return nil
			}
			for _, wf := range wfs {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s v%d (%s)\n", wf.Name, wf.Version, wf.SourcePath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	return cmd
}

func newWorkflowShowCmd() *cobra.Command {
	var (
		team string
		name string
	)
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show workflow entries for a name (all versions)",
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

			wfs, err := st.ListWorkflows(cmd.Context(), team)
			if err != nil {
				return err
			}
			found := false
			for _, wf := range wfs {
				if wf.Name == name {
					found = true
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s v%d source=%s\n", wf.Name, wf.Version, wf.SourcePath)
				}
			}
			if !found {
				return fmt.Errorf("workflow not found: %s", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().StringVar(&name, "name", "", "Workflow name")
	return cmd
}
