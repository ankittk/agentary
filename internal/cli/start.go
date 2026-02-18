package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/daemon"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var (
		port           int
		foreground     bool
		intervalSec    float64
		maxConcurrent  int
		dev            bool
		pprofAddr      string
		runtimeKind    string
		subprocessCmd  string
		subprocessArgs []string
		grpcAddr       string
		envFile        string
		sandboxHome    string
		dbDriver       string
		dbURL          string
		enableOtel     bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Agentary (web UI + daemon loop)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envFile != "" {
				if err := loadEnvFile(envFile); err != nil {
					return err
				}
			}
			home := config.MustHomeFrom(cmd.Context())

			opts := daemon.StartOptions{
				Home:           home,
				Port:           port,
				IntervalSec:    intervalSec,
				MaxConcurrent:  maxConcurrent,
				Dev:            dev,
				PprofAddr:      pprofAddr,
				Runtime:        runtimeKind,
				SubprocessCmd:  subprocessCmd,
				SubprocessArgs: subprocessArgs,
				GrpcAddr:       grpcAddr,
				SandboxHome:    sandboxHome,
				DBDriver:       dbDriver,
				DBURL:          dbURL,
				EnableOtel:     enableOtel,
			}

			ui := (&url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port)}).String()

			if foreground {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Starting Agentary in foreground on %s\n", ui)
				return daemon.StartForeground(cmd.Context(), opts)
			}

			pid, err := daemon.StartBackground(cmd.Context(), opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Agentary started (pid %d)\n", pid)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "UI: %s\n", ui)

			// Best-effort open browser (Linux: xdg-open, macOS: open, Windows: start).
			_ = openBrowser(ui)
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 3548, "Port for the web UI")
	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground (do not daemonize)")
	cmd.Flags().Float64Var(&intervalSec, "interval", 1.0, "Scheduler poll interval (seconds)")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 32, "Max concurrent agent turns")
	cmd.Flags().BoolVar(&dev, "dev", false, "Enable dev mode")
	cmd.Flags().StringVar(&pprofAddr, "pprof", "", "Enable pprof on address (e.g. 127.0.0.1:6060)")
	cmd.Flags().StringVar(&runtimeKind, "runtime", "stub", "Runtime: stub, subprocess, or grpc")
	cmd.Flags().StringVar(&subprocessCmd, "subprocess-cmd", "", "Command for subprocess runtime (e.g. agent-runner)")
	cmd.Flags().StringSliceVar(&subprocessArgs, "subprocess-args", nil, "Args for subprocess runtime")
	cmd.Flags().StringVar(&grpcAddr, "grpc-addr", "", "Agent gRPC server address for runtime=grpc (e.g. localhost:50051)")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Load env vars from file (KEY=VALUE per line) before starting")
	cmd.Flags().StringVar(&sandboxHome, "sandbox-home", "", "Run subprocess inside bubblewrap with this dir writable (Linux only)")
	cmd.Flags().StringVar(&dbDriver, "db-driver", "sqlite", "Store driver: sqlite or postgres")
	cmd.Flags().StringVar(&dbURL, "db-url", "", "DB connection string (for postgres; or set DATABASE_URL)")
	cmd.Flags().BoolVar(&enableOtel, "otel", true, "Enable OpenTelemetry metrics (Prometheus exporter, HTTP/SSE/task/agent instrumentation)")

	return cmd
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		value := strings.TrimSpace(line[i+1:])
		if key != "" {
			_ = os.Setenv(key, value)
		}
	}
	return sc.Err()
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", u).Start()
	default:
		// Linux and others
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return err
		}
		return exec.Command("xdg-open", u).Start()
	}
}
