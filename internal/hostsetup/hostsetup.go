package hostsetup

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Options struct {
	HomeDir    string
	ProjectDir string
	BinaryPath string
	Apply      bool
}

type Result struct {
	OK         bool         `json:"ok"`
	Applied    bool         `json:"applied"`
	BinaryPath string       `json:"binary_path"`
	ProjectDir string       `json:"project_dir"`
	Hosts      []HostResult `json:"hosts"`
	Warnings   []string     `json:"warnings"`
}

type HostResult struct {
	Name       string   `json:"name"`
	ConfigPath string   `json:"config_path"`
	Changed    bool     `json:"changed"`
	Action     string   `json:"action"`
	Command    string   `json:"command"`
	Args       []string `json:"args"`
}

func Setup(options Options) (Result, error) {
	resolved, err := resolveOptions(options)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		OK:         true,
		Applied:    resolved.Apply,
		BinaryPath: resolved.BinaryPath,
		ProjectDir: resolved.ProjectDir,
		Warnings:   []string{},
	}
	hosts := []struct {
		name string
		path string
		next func() (string, error)
	}{
		{name: "codex", path: filepath.Join(resolved.HomeDir, ".codex", "config.toml"), next: func() (string, error) {
			current := readOptional(filepath.Join(resolved.HomeDir, ".codex", "config.toml"))
			return upsertTOMLTable(current, "[mcp_servers.llm-wiki]", codexBlock(resolved.BinaryPath)), nil
		}},
		{name: "claude", path: filepath.Join(resolved.ProjectDir, ".mcp.json"), next: func() (string, error) {
			return upsertMCPJSON(readOptional(filepath.Join(resolved.ProjectDir, ".mcp.json")), resolved.BinaryPath)
		}},
		{name: "reasonix", path: filepath.Join(resolved.ProjectDir, "reasonix.toml"), next: func() (string, error) {
			current := readOptional(filepath.Join(resolved.ProjectDir, "reasonix.toml"))
			return upsertReasonixPlugin(current, reasonixBlock(resolved.BinaryPath)), nil
		}},
	}
	for _, host := range hosts {
		next, err := host.next()
		if err != nil {
			return Result{}, err
		}
		current := readOptional(host.path)
		changed := current != next
		action := "unchanged"
		if changed {
			action = "would_write"
			if resolved.Apply {
				action = "wrote"
				if err := writeTextFile(host.path, next); err != nil {
					return Result{}, err
				}
			}
		}
		result.Hosts = append(result.Hosts, HostResult{
			Name:       host.name,
			ConfigPath: host.path,
			Changed:    changed,
			Action:     action,
			Command:    resolved.BinaryPath,
			Args:       []string{"mcp"},
		})
	}
	return result, nil
}

func resolveOptions(options Options) (Options, error) {
	if options.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Options{}, err
		}
		options.HomeDir = home
	}
	if options.ProjectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Options{}, err
		}
		options.ProjectDir = cwd
	}
	var err error
	options.HomeDir, err = filepath.Abs(options.HomeDir)
	if err != nil {
		return Options{}, err
	}
	options.ProjectDir, err = filepath.Abs(options.ProjectDir)
	if err != nil {
		return Options{}, err
	}
	if options.BinaryPath == "" {
		if path, err := exec.LookPath("llm-wiki"); err == nil {
			options.BinaryPath = path
		} else {
			options.BinaryPath, err = os.Executable()
			if err != nil {
				return Options{}, err
			}
		}
	}
	options.BinaryPath, err = filepath.Abs(options.BinaryPath)
	if err != nil {
		return Options{}, err
	}
	return options, nil
}

func codexBlock(binary string) string {
	return `[mcp_servers.llm-wiki]
command = "` + filepath.ToSlash(binary) + `"
args = ["mcp"]
startup_timeout_sec = 10
tool_timeout_sec = 60
`
}

func reasonixBlock(binary string) string {
	return `[[plugins]]
name = "llm-wiki"
type = "stdio"
command = "` + filepath.ToSlash(binary) + `"
args = ["mcp"]
`
}

func upsertMCPJSON(current string, binary string) (string, error) {
	root := map[string]any{}
	if strings.TrimSpace(current) != "" {
		if err := json.Unmarshal([]byte(current), &root); err != nil {
			return "", err
		}
	}
	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		servers = map[string]any{}
		root["mcpServers"] = servers
	}
	servers["llm-wiki"] = map[string]any{
		"command": filepath.ToSlash(binary),
		"args":    []string{"mcp"},
		"env":     map[string]any{},
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

func upsertTOMLTable(current string, header string, block string) string {
	lines := strings.Split(current, "\n")
	var out []string
	replaced := false
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) == header {
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			out = appendBlock(out, block)
			replaced = true
			i++
			for i < len(lines) && !isTOMLHeader(lines[i]) {
				i++
			}
			continue
		}
		out = append(out, lines[i])
		i++
	}
	if !replaced {
		out = appendBlock(out, block)
	}
	return cleanTrailingBlankLines(out)
}

func upsertReasonixPlugin(current string, block string) string {
	lines := strings.Split(current, "\n")
	var out []string
	replaced := false
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) == "[[plugins]]" {
			start := i
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != "[[plugins]]" {
				i++
			}
			chunk := strings.Join(lines[start:i], "\n")
			if strings.Contains(chunk, `name = "llm-wiki"`) {
				if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
					out = out[:len(out)-1]
				}
				out = appendBlock(out, block)
				replaced = true
				continue
			}
			out = append(out, lines[start:i]...)
			continue
		}
		out = append(out, lines[i])
		i++
	}
	if !replaced {
		out = appendBlock(out, block)
	}
	return cleanTrailingBlankLines(out)
}

func appendBlock(out []string, block string) []string {
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	block = strings.TrimRight(block, "\n")
	return append(out, strings.Split(block, "\n")...)
}

func isTOMLHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func cleanTrailingBlankLines(lines []string) string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func readOptional(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func writeTextFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
