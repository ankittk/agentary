package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ankittk/agentary/internal/sandbox"
)

// SubprocessRuntime runs a local agent binary: stdin = JSON TurnRequest, stdout = NDJSON events per line.
// If SandboxHome is set (and bubblewrap is available on Linux), the process runs inside a minimal bwrap sandbox.
// If SandboxTeamDir is also set (must be under SandboxHome), only that directory is writable; SandboxHome
// (including protected/) is read-only.
type SubprocessRuntime struct {
	Command         string
	Args            []string
	Timeout         time.Duration // 0 = use context only
	SandboxHome     string        // if set, run agent inside bubblewrap with this dir writable
	SandboxTeamDir  string        // if set with SandboxHome, restrict writes to this dir only (team dir)
}

func (r SubprocessRuntime) Name() string { return "subprocess" }

func (r SubprocessRuntime) RunTurn(ctx context.Context, req TurnRequest, emit func(Event)) (TurnResult, error) {
	if r.Command == "" {
		return TurnResult{}, errors.New("subprocess command is required")
	}
	var cmd *exec.Cmd
	if r.SandboxHome != "" {
		cmd = sandbox.WrapCommand(ctx, r.SandboxHome, r.SandboxTeamDir, r.Command, r.Args)
	} else {
		cmd = exec.CommandContext(ctx, r.Command, r.Args...)
	}
	// Pass network allowlist via env so the agent binary can enforce egress (see docs/content/sandboxing.md).
	if len(req.NetworkAllowlist) > 0 {
		cmd.Env = append(os.Environ(), "AGENTARY_NETWORK_ALLOWLIST="+strings.Join(req.NetworkAllowlist, ","))
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return TurnResult{}, err
	}
	cmd.Stdin = strings.NewReader(string(reqJSON) + "\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return TurnResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return TurnResult{}, err
	}
	defer func() {
		if ctx.Err() != nil {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
		if err := cmd.Wait(); err != nil {
			slog.Warn("subprocess exited with error", "err", err)
		}
	}()

	var output strings.Builder
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}
		if ev.Timestamp.IsZero() {
			ev.Timestamp = time.Now().UTC()
		}
		emit(ev)
	}
	if err := sc.Err(); err != nil {
		return TurnResult{}, err
	}
	return TurnResult{Output: strings.TrimSpace(output.String())}, nil
}
