package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newApikeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "Generate API key for protecting the server when exposed over a network",
	}
	cmd.AddCommand(newApikeyGenerateCmd())
	return cmd
}

func newApikeyGenerateCmd() *cobra.Command {
	var envFile string
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a random API key and print usage instructions",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("generate key: %w", err)
			}
			key := hex.EncodeToString(b)

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Generated API key (save it somewhere safe):")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  "+key)
			fmt.Fprintln(out)

			if envFile != "" {
				line := "AGENTARY_API_KEY=" + key + "\n"
				f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
				if err != nil {
					return fmt.Errorf("write %s: %w", envFile, err)
				}
				if _, err := f.WriteString(line); err != nil {
					_ = f.Close()
					return fmt.Errorf("write %s: %w", envFile, err)
				}
				if err := f.Close(); err != nil {
					return err
				}
				fmt.Fprintf(out, "Appended AGENTARY_API_KEY to %s\n", envFile)
				fmt.Fprintln(out, "Start the server with: agentary start --foreground --env-file "+envFile)
			} else {
				fmt.Fprintln(out, "Use it:")
				fmt.Fprintln(out, "  1. On the server: export AGENTARY_API_KEY="+key)
				fmt.Fprintln(out, "     Or add to .env and run: agentary start --foreground --env-file .env")
				fmt.Fprintln(out, "  2. In clients: send header X-API-Key: <key> or query ?api_key=<key>")
			}
			fmt.Fprintln(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&envFile, "env", "", "Append AGENTARY_API_KEY to this file (e.g. .env)")
	return cmd
}
