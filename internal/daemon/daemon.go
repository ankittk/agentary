package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ankittk/agentary/internal/httpapi"
	"github.com/ankittk/agentary/internal/manager"
	"github.com/ankittk/agentary/internal/merge"
	"github.com/ankittk/agentary/internal/otel"
	"github.com/ankittk/agentary/internal/store"
)

var errNotRunning = errors.New("agentary is not running")

func StartForeground(ctx context.Context, opts StartOptions) error {
	if opts.Home == "" {
		return errors.New("home is required")
	}
	if opts.Port == 0 {
		opts.Port = 3548
	}

	// Ensure dirs exist.
	if err := os.MkdirAll(protectedDir(opts.Home), 0o755); err != nil {
		return err
	}

	// Acquire singleton lock (released on exit).
	lock, err := acquireLock(lockPath(opts.Home))
	if err != nil {
		return err
	}
	defer lock.release()

	// Optional pprof.
	startPprof(opts.PprofAddr)

	// Ensure DB schema exists before serving (SQLite only; Postgres migrates on connect).
	if opts.DBDriver != "postgres" {
		if err := store.EnsureSchema(opts.Home); err != nil {
			return err
		}
	}

	// Write PID + addr files.
	pid := os.Getpid()
	if err := os.WriteFile(pidPath(opts.Home), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return err
	}
	addr := fmt.Sprintf("0.0.0.0:%d", opts.Port)
	_ = os.WriteFile(addrPath(opts.Home), []byte(addr+"\n"), 0o644)
	defer func() {
		_ = os.Remove(pidPath(opts.Home))
		_ = os.Remove(addrPath(opts.Home))
	}()

	// Early port check for clearer error.
	if err := checkPortAvailable(opts.Port); err != nil {
		return err
	}

	// Manager LLM from env if not set in opts
	if opts.ManagerLLMURL == "" {
		opts.ManagerLLMURL = os.Getenv("AGENTARY_LLM_URL")
	}
	if opts.ManagerLLMKey == "" {
		opts.ManagerLLMKey = os.Getenv("OPENAI_API_KEY")
	}
	if opts.ManagerLLMModel == "" {
		opts.ManagerLLMModel = os.Getenv("AGENTARY_LLM_MODEL")
		if opts.ManagerLLMModel == "" {
			opts.ManagerLLMModel = "gpt-4o-mini"
		}
	}
	srvOpts := httpapi.ServerOptions{
		Home:     opts.Home,
		Addr:     addr,
		Dev:      opts.Dev,
		APIKey:   os.Getenv("AGENTARY_API_KEY"),
		DBDriver: opts.DBDriver,
		DBURL:    opts.DBURL,
	}
	if opts.EnableOtel {
		metricsHandler, err := otel.InitMeterProvider(ctx, "agentary")
		if err != nil {
			slog.Warn("otel init failed, using legacy metrics", "err", err)
		} else {
			srvOpts.MetricsHandler = metricsHandler
			srvOpts.UseOtelHTTP = true
		}
	}
	app, err := httpapi.NewApp(srvOpts)
	if err != nil {
		return err
	}
	if opts.EnableOtel {
		_ = otel.InitMetricsWithTaskCount(ctx, func() (todo, inProgress, done, failed int64) {
			teams, _ := app.Store.ListTeams(context.Background())
			for _, t := range teams {
				tasks, _ := app.Store.ListTasks(context.Background(), t.Name, 0)
				for _, tk := range tasks {
					switch tk.Status {
					case "todo":
						todo++
					case "in_progress":
						inProgress++
					case "done":
						done++
					case "failed":
						failed++
					}
				}
			}
			return todo, inProgress, done, failed
		})
	}

	slog.Info("daemon starting", "addr", addr, "home", opts.Home)
	errCh := make(chan error, 1)
	go func() {
		// Scheduler runs alongside the HTTP server and publishes SSE events.
		go runScheduler(ctx, opts, app)
		// Merge worker processes tasks in Merging stage (rebase, test, merge, clean).
		go (&merge.Worker{Store: app.Store, RebaseBeforeMerge: opts.RebaseBeforeMerge}).Run(ctx)
		// Manager: LLM-backed if AGENTARY_LLM_URL + OPENAI_API_KEY set, else rule-based.
		if opts.ManagerLLMURL != "" && opts.ManagerLLMKey != "" {
			go manager.RunLLM(ctx, app, manager.LLMOpts{
				BaseURL: opts.ManagerLLMURL,
				APIKey:  opts.ManagerLLMKey,
				Model:   opts.ManagerLLMModel,
			})
		} else {
			go manager.Run(ctx, app)
		}
		// Poll message inbox for "manager" to drive turns (e.g. /shell, create task from message).
		go manager.PollInbox(ctx, app, "manager", 5*time.Second)
		errCh <- app.Server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = app.Server.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func StartBackground(ctx context.Context, opts StartOptions) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	// Ensure dirs exist before starting.
	if err := os.MkdirAll(protectedDir(opts.Home), 0o755); err != nil {
		return 0, err
	}

	// Best-effort: refuse to start if already running.
	if st, _ := Status(ctx, opts.Home); st.Running {
		return 0, fmt.Errorf("agentary already running (pid %d)", st.PID)
	}

	logFile := filepath.Join(protectedDir(opts.Home), "daemon.log")
	stderr, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	// Kept open for child lifetime; closing here may break writes on some platforms.

	args := []string{
		"daemon",
		"--home", opts.Home,
		"--port", strconv.Itoa(opts.Port),
		"--interval", fmt.Sprintf("%g", opts.IntervalSec),
		"--max-concurrent", strconv.Itoa(opts.MaxConcurrent),
	}
	if opts.Dev {
		args = append(args, "--dev")
	}
	if opts.PprofAddr != "" {
		args = append(args, "--pprof", opts.PprofAddr)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	// Wait briefly for pid file to appear or process to die.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st, _ := Status(ctx, opts.Home); st.Running {
			return st.PID, nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Fallback to started pid even if status isn't ready yet.
	return cmd.Process.Pid, nil
}

func Stop(ctx context.Context, home string) (bool, error) {
	st, err := Status(ctx, home)
	if err != nil {
		return false, err
	}
	if !st.Running {
		return false, nil
	}

	proc, err := os.FindProcess(st.PID)
	if err != nil {
		// On unix FindProcess always succeeds; keep this for completeness.
		return false, errNotRunning
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return false, err
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if st2, _ := Status(ctx, home); !st2.Running {
			return true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = proc.Kill()
	return true, nil
}

func Status(ctx context.Context, home string) (StatusInfo, error) {
	pb, err := os.ReadFile(pidPath(home))
	if err != nil {
		return StatusInfo{Running: false}, nil
	}
	pidStr := strings.TrimSpace(string(pb))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return StatusInfo{Running: false}, nil
	}

	// kill(pid, 0) checks existence/permission on unix.
	if err := syscall.Kill(pid, 0); err != nil {
		_ = os.Remove(pidPath(home))
		return StatusInfo{Running: false}, nil
	}

	addr := ""
	if ab, err := os.ReadFile(addrPath(home)); err == nil {
		addr = strings.TrimSpace(string(ab))
	}
	if addr == "" {
		addr = "unknown"
	}
	return StatusInfo{Running: true, PID: pid, Addr: addr}, nil
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return fmt.Errorf("port %d is already in use", port)
	}
	_ = ln.Close()
	return nil
}
