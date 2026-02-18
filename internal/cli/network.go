package cli

import (
	"errors"
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage network allowlist",
	}
	cmd.AddCommand(newNetworkShowCmd())
	cmd.AddCommand(newNetworkAllowCmd())
	cmd.AddCommand(newNetworkDisallowCmd())
	cmd.AddCommand(newNetworkResetCmd())
	return cmd
}

func newNetworkShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show network allowlist",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			domains, err := st.ListAllowedDomains(cmd.Context())
			if err != nil {
				return err
			}
			if len(domains) == 1 && domains[0] == "*" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Network allowlist: * (unrestricted)")
				return nil
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Network allowlist:")
			for _, d := range domains {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", d)
			}
			return nil
		},
	}
	return cmd
}

func newNetworkAllowCmd() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "allow",
		Short: "Add a domain to the allowlist (or '*' for unrestricted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return errors.New("--domain is required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.AllowDomain(cmd.Context(), domain); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain to allow (api.github.com, *.example.com, or '*')")
	return cmd
}

func newNetworkDisallowCmd() *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "disallow",
		Short: "Remove a domain from the allowlist",
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return errors.New("--domain is required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.DisallowDomain(cmd.Context(), domain); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain to disallow")
	return cmd
}

func newNetworkResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset allowlist to '*' (unrestricted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.ResetAllowlist(cmd.Context()); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Reset to *.")
			return nil
		},
	}
	return cmd
}
