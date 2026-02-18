package cli

import (
	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/daemon"
	"github.com/spf13/cobra"
)

func newDaemonCmd() *cobra.Command {
	var (
		port           int
		intervalSec    float64
		maxConcurrent  int
		dev            bool
		pprofAddr      string
		runtimeKind    string
		subprocessCmd  string
		subprocessArgs []string
		grpcAddr       string
		enableOtel     bool
	)

	cmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Internal: run daemon process",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			home := config.MustHomeFrom(cmd.Context())
			return daemon.StartForeground(cmd.Context(), daemon.StartOptions{
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
				EnableOtel:     enableOtel,
			})
		},
	}

	cmd.Flags().IntVar(&port, "port", 3548, "Port for the web UI")
	cmd.Flags().Float64Var(&intervalSec, "interval", 1.0, "Scheduler poll interval (seconds)")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 32, "Max concurrent agent turns")
	cmd.Flags().BoolVar(&dev, "dev", false, "Enable dev mode")
	cmd.Flags().StringVar(&pprofAddr, "pprof", "", "Enable pprof on address (e.g. 127.0.0.1:6060)")
	cmd.Flags().StringVar(&runtimeKind, "runtime", "stub", "Runtime: stub, subprocess, or grpc")
	cmd.Flags().StringVar(&subprocessCmd, "subprocess-cmd", "", "Command for subprocess runtime")
	cmd.Flags().StringSliceVar(&subprocessArgs, "subprocess-args", nil, "Args for subprocess runtime")
	cmd.Flags().StringVar(&grpcAddr, "grpc-addr", "", "Agent gRPC server address for runtime=grpc")
	cmd.Flags().BoolVar(&enableOtel, "otel", true, "Enable OpenTelemetry metrics")

	return cmd
}
