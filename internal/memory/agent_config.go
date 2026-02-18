package memory

import (
	"os"

	"gopkg.in/yaml.v3"
)

// AgentConfig holds per-agent model settings (e.g. model name, max tokens).
type AgentConfig struct {
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// LoadAgentConfig loads config from <agentDir>/config.yaml. Returns nil config and nil error if file is missing.
func LoadAgentConfig(agentDir string) (*AgentConfig, error) {
	path := AgentConfigPath(agentDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveAgentConfig writes the agent config to <agentDir>/config.yaml.
func SaveAgentConfig(agentDir string, cfg *AgentConfig) error {
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(AgentConfigPath(agentDir), data, 0o644)
}
