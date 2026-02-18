package daemon

// StartOptions configures the daemon (home, port, scheduler interval, runtime, DB, manager LLM, etc.).
type StartOptions struct {
	Home           string
	Port           int
	IntervalSec    float64
	MaxConcurrent  int
	Dev            bool
	PprofAddr      string
	Runtime        string   // "stub", "subprocess", or "grpc"
	SubprocessCmd  string   // e.g. "agent-runner"
	SubprocessArgs []string // e.g. ["--config", "default"]
	GrpcAddr       string   // for runtime=grpc: agent gRPC server address (e.g. "localhost:50051")
	SandboxHome    string   // if set, run subprocess inside bubblewrap with this dir writable (Linux only)
	DBDriver       string   // "sqlite" (default) or "postgres"
	DBURL          string   // for postgres: connection string (or DATABASE_URL env)
	// Manager LLM: when both set, use LLM manager instead of rule-based.
	ManagerLLMURL     string // e.g. https://api.openai.com
	ManagerLLMKey     string // OPENAI_API_KEY
	ManagerLLMModel   string // e.g. gpt-4o-mini
	RebaseBeforeMerge bool   // merge worker rebases task branch onto main before merging
	EnableOtel        bool   // enable OpenTelemetry metrics (Prometheus exporter + HTTP/SSE/task/agent instrumentation)
}

// StatusInfo is the result of Status (running or not, PID, listen addr).
type StatusInfo struct {
	Running bool
	PID     int
	Addr    string
}
