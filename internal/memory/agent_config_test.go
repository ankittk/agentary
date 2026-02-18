package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentConfig_SaveAgentConfig(t *testing.T) {
	t.Parallel()
	agentDir := filepath.Join(t.TempDir(), "agents", "a1")

	// Missing file returns nil, nil
	cfg, err := LoadAgentConfig(agentDir)
	if err != nil {
		t.Fatalf("LoadAgentConfig missing: %v", err)
	}
	if cfg != nil {
		t.Fatalf("LoadAgentConfig missing: expected nil, got %+v", cfg)
	}

	// Save then load
	saved := &AgentConfig{Model: "gpt-4", MaxTokens: 1024}
	if err := SaveAgentConfig(agentDir, saved); err != nil {
		t.Fatalf("SaveAgentConfig: %v", err)
	}
	cfg, err = LoadAgentConfig(agentDir)
	if err != nil {
		t.Fatalf("LoadAgentConfig: %v", err)
	}
	if cfg == nil || cfg.Model != "gpt-4" || cfg.MaxTokens != 1024 {
		t.Fatalf("LoadAgentConfig: got %+v", cfg)
	}
}

func TestLoadAgentConfig_invalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "a1")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := AgentConfigPath(agentDir)
	if err := os.WriteFile(path, []byte("not: valid: yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAgentConfig(agentDir)
	if err == nil {
		t.Fatal("LoadAgentConfig: expected error for invalid YAML")
	}
}
