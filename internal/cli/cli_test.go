package cli

import (
	"bytes"
	"regexp"
	"testing"
)

func TestNewRootCmd_hasSubcommands(t *testing.T) {
	root := NewRootCmd("test")
	if root == nil {
		t.Fatal("NewRootCmd returned nil")
	}
	cmds := root.Commands()
	names := make(map[string]bool)
	for _, c := range cmds {
		names[c.Name()] = true
	}
	for _, want := range []string{"start", "stop", "status", "team", "agent", "workflow", "network", "apikey"} {
		if !names[want] {
			t.Errorf("expected subcommand %q", want)
		}
	}
}

func TestNewRootCmd_versionFlag(t *testing.T) {
	root := NewRootCmd("1.2.3")
	if root.Version != "1.2.3" {
		t.Errorf("Version: got %q", root.Version)
	}
}

func TestNewRootCmd_hasHomeFlag(t *testing.T) {
	root := NewRootCmd("")
	f := root.PersistentFlags().Lookup("home")
	if f == nil {
		t.Fatal("expected --home persistent flag")
	}
}

func TestApikeyGenerate(t *testing.T) {
	root := NewRootCmd("")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"apikey", "generate"})
	if err := root.Execute(); err != nil {
		t.Fatalf("apikey generate: %v", err)
	}
	out := buf.String()
	hexKey := regexp.MustCompile(`(?m)^  ([a-f0-9]{64})$`)
	if !hexKey.MatchString(out) {
		t.Errorf("output should contain a 64-char hex key on its own line; got:\n%s", out)
	}
	if !regexp.MustCompile(`AGENTARY_API_KEY`).MatchString(out) {
		t.Errorf("output should mention AGENTARY_API_KEY")
	}
	if !regexp.MustCompile(`X-API-Key`).MatchString(out) {
		t.Errorf("output should mention X-API-Key")
	}
}
