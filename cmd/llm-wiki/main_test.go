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
		OK        bool   `json:"ok"`
		Applied   bool   `json:"applied"`
		VaultPath string `json:"vault_path"`
		Hosts     []struct {
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
	wantVault := filepath.Join(home, "workspace", "knowledge-base", "llm-wiki")
	if result.VaultPath != wantVault {
		t.Fatalf("VaultPath = %q, want default %q", result.VaultPath, wantVault)
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

func TestValidateCommandDefaultsToConfiguredVault(t *testing.T) {
	t.Setenv("LLM_WIKI_VAULT", "../../fixtures/okf-minimal")
	var stdout bytes.Buffer
	cmd := rootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"validate", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var result struct {
		OK           bool   `json:"ok"`
		BundleRoot   string `json:"bundle_root"`
		ConceptCount int    `json:"concept_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.BundleRoot == "" || result.ConceptCount != 1 {
		t.Fatalf("result = %#v, want vault-backed valid OKF bundle", result)
	}
}

func TestSetupHostsCommandAcceptsVaultPath(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	vault := filepath.Join(home, "knowledge-base", "llm-wiki")
	bin := filepath.Join(home, "bin", "llm-wiki")
	var stdout bytes.Buffer
	cmd := rootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"setup-hosts", "--json", "--home", home, "--project", project, "--bin", bin, "--vault", vault})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var result struct {
		OK        bool   `json:"ok"`
		VaultPath string `json:"vault_path"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.VaultPath != vault {
		t.Fatalf("result = %#v, want configured vault path %q", result, vault)
	}
}

func TestSetupHostsCommandPromptsForVaultPath(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	vault := filepath.Join(home, "custom-vault")
	bin := filepath.Join(home, "bin", "llm-wiki")
	var stdout bytes.Buffer
	cmd := rootCmd()
	cmd.SetIn(bytes.NewBufferString(vault + "\n"))
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"setup-hosts", "--apply", "--home", home, "--project", project, "--bin", bin})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	config, err := os.ReadFile(filepath.Join(project, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(config, []byte(`"LLM_WIKI_VAULT": "`+filepath.ToSlash(vault)+`"`)) {
		t.Fatalf("claude config missing prompted vault path:\n%s", config)
	}
}
