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
			_, _ = fmt.Fprintln(out, "Generated API key (save it somewhere safe):")
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "  "+key)
			_, _ = fmt.Fprintln(out)

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
				_, _ = fmt.Fprintf(out, "Appended AGENTARY_API_KEY to %s\n", envFile)
				_, _ = fmt.Fprintln(out, "Start the server with: agentary start --foreground --env-file "+envFile)
			} else {
				_, _ = fmt.Fprintln(out, "Use it:")
				_, _ = fmt.Fprintln(out, "  1. On the server: export AGENTARY_API_KEY="+key)
				_, _ = fmt.Fprintln(out, "     Or add to .env and run: agentary start --foreground --env-file .env")
				_, _ = fmt.Fprintln(out, "  2. In clients: send header X-API-Key: <key> or query ?api_key=<key>")
			}
			_, _ = fmt.Fprintln(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&envFile, "env", "", "Append AGENTARY_API_KEY to this file (e.g. .env)")
	return cmd
}
