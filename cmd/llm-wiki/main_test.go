package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupHostsCommandDefaultsToDryRun(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	bin := filepath.Join(home, "bin", "llm-wiki")
	var stdout bytes.Buffer
	cmd := rootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"setup-hosts", "--json", "--home", home, "--project", project, "--bin", bin})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var result struct {
		OK      bool `json:"ok"`
		Applied bool `json:"applied"`
		Hosts   []struct {
			Name string `json:"name"`
		} `json:"hosts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatal("OK = false, want true")
	}
	if result.Applied {
		t.Fatal("Applied = true, want dry-run")
	}
	if len(result.Hosts) != 3 {
		t.Fatalf("hosts = %d, want 3", len(result.Hosts))
	}
	for _, path := range []string{
		filepath.Join(home, ".codex", "config.toml"),
		filepath.Join(project, ".mcp.json"),
		filepath.Join(project, "reasonix.toml"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s exists after dry-run or stat failed: %v", path, err)
		}
	}
}
