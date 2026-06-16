package hostsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupDryRunPlansAllHostsWithoutWriting(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	bin := filepath.Join(home, "bin", "llm-wiki")

	result, err := Setup(Options{
		HomeDir:    home,
		ProjectDir: project,
		BinaryPath: bin,
		Apply:      false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.OK {
		t.Fatal("OK = false, want true")
	}
	if result.Applied {
		t.Fatal("Applied = true, want false")
	}
	if len(result.Hosts) != 3 {
		t.Fatalf("hosts = %d, want 3", len(result.Hosts))
	}
	for _, host := range []string{"codex", "claude", "reasonix"} {
		if !hasHost(result, host) {
			t.Fatalf("missing host result %q: %+v", host, result.Hosts)
		}
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

func TestSetupApplyWritesHostConfigsWithoutRemovingLegacyCodexPlugin(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	bin := filepath.Join(home, "bin", "llm-wiki")
	codexConfig := filepath.Join(home, ".codex", "config.toml")
	writeFile(t, codexConfig, `[plugins."llm-wiki@llm-wiki-marketplace"]
enabled = true

[hooks.state."wiki@llm-wiki:hooks/hooks.json:session_start:0:0"]
trusted_hash = "sha256:old"

[mcp_servers.llm-wiki]
command = "old-llm-wiki"
args = ["mcp"]

[mcp_servers.other]
command = "other"
`)
	legacyCache := filepath.Join(home, ".codex", "plugins", "cache", "llm-wiki-marketplace")
	if err := os.MkdirAll(legacyCache, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Setup(Options{
		HomeDir:    home,
		ProjectDir: project,
		BinaryPath: bin,
		Apply:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.OK || !result.Applied {
		t.Fatalf("result = %+v, want ok applied", result)
	}
	config := readFile(t, codexConfig)
	for _, want := range []string{
		`[mcp_servers.llm-wiki]`,
		`command = "` + filepath.ToSlash(bin) + `"`,
		`args = ["mcp"]`,
		`startup_timeout_sec = 10`,
		`tool_timeout_sec = 60`,
		`[mcp_servers.other]`,
	} {
		if !strings.Contains(filepath.ToSlash(config), want) {
			t.Fatalf("codex config missing %q:\n%s", want, config)
		}
	}
	for _, wantPreserved := range []string{`[plugins."llm-wiki@llm-wiki-marketplace"]`, `wiki@llm-wiki`} {
		if !strings.Contains(config, wantPreserved) {
			t.Fatalf("codex config did not preserve %q:\n%s", wantPreserved, config)
		}
	}
	for _, forbidden := range []string{`command = "old-llm-wiki"`} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("codex config still contains %q:\n%s", forbidden, config)
		}
	}

	claudeConfig := readFile(t, filepath.Join(project, ".mcp.json"))
	if !strings.Contains(claudeConfig, `"llm-wiki"`) || !strings.Contains(filepath.ToSlash(claudeConfig), filepath.ToSlash(bin)) {
		t.Fatalf("claude mcp config not written correctly:\n%s", claudeConfig)
	}
	reasonixConfig := readFile(t, filepath.Join(project, "reasonix.toml"))
	if !strings.HasPrefix(reasonixConfig, "[[plugins]]") {
		t.Fatalf("reasonix config should not start with a blank line:\n%s", reasonixConfig)
	}
	if !strings.Contains(reasonixConfig, `name = "llm-wiki"`) || !strings.Contains(filepath.ToSlash(reasonixConfig), filepath.ToSlash(bin)) {
		t.Fatalf("reasonix config not written correctly:\n%s", reasonixConfig)
	}
	if _, err := os.Stat(legacyCache); err != nil {
		t.Fatalf("legacy cache should be left untouched: %v", err)
	}
}

func TestSetupApplyWritesConfiguredVaultToHostConfigs(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	vault := filepath.Join(home, "knowledge-base", "llm-wiki")
	bin := filepath.Join(home, "bin", "llm-wiki")

	result, err := Setup(Options{
		HomeDir:    home,
		ProjectDir: project,
		BinaryPath: bin,
		VaultPath:  vault,
		Apply:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VaultPath != vault {
		t.Fatalf("VaultPath = %q, want %q", result.VaultPath, vault)
	}

	codexConfig := filepath.ToSlash(readFile(t, filepath.Join(home, ".codex", "config.toml")))
	for _, want := range []string{
		`[mcp_servers.llm-wiki.env]`,
		`LLM_WIKI_VAULT = "` + filepath.ToSlash(vault) + `"`,
	} {
		if !strings.Contains(codexConfig, want) {
			t.Fatalf("codex config missing %q:\n%s", want, codexConfig)
		}
	}

	claudeConfig := filepath.ToSlash(readFile(t, filepath.Join(project, ".mcp.json")))
	if !strings.Contains(claudeConfig, `"LLM_WIKI_VAULT": "`+filepath.ToSlash(vault)+`"`) {
		t.Fatalf("claude config missing vault env:\n%s", claudeConfig)
	}

	reasonixConfig := filepath.ToSlash(readFile(t, filepath.Join(project, "reasonix.toml")))
	if !strings.Contains(reasonixConfig, `LLM_WIKI_VAULT = "`+filepath.ToSlash(vault)+`"`) {
		t.Fatalf("reasonix config missing vault env:\n%s", reasonixConfig)
	}
}

func TestSetupDefaultsVaultPathFromHomeDir(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	bin := filepath.Join(home, "bin", "llm-wiki")
	wantVault := filepath.Join(home, "workspace", "knowledge-base", "llm-wiki")

	result, err := Setup(Options{
		HomeDir:    home,
		ProjectDir: project,
		BinaryPath: bin,
		Apply:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VaultPath != wantVault {
		t.Fatalf("VaultPath = %q, want default %q", result.VaultPath, wantVault)
	}

	codexConfig := filepath.ToSlash(readFile(t, filepath.Join(home, ".codex", "config.toml")))
	if !strings.Contains(codexConfig, `LLM_WIKI_VAULT = "`+filepath.ToSlash(wantVault)+`"`) {
		t.Fatalf("codex config missing default vault env:\n%s", codexConfig)
	}
}

func hasHost(result Result, name string) bool {
	for _, host := range result.Hosts {
		if host.Name == name {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
