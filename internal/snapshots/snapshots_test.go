package snapshots

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIGoldenSnapshots(t *testing.T) {
	repo := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "llm-wiki")
	build := exec.Command("go", "build", "-o", bin, "./cmd/llm-wiki")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	cases := []struct {
		name string
		args []string
		env  []string
		code int
	}{
		{
			name: "validate-minimal",
			args: []string{"validate", "fixtures/okf-minimal", "--json"},
			code: 0,
		},
		{
			name: "validate-invalid-missing-type",
			args: []string{"validate", "fixtures/okf-invalid-missing-type", "--json"},
			code: 1,
		},
		{
			name: "query-pack-alpha",
			args: []string{"query-pack", "fixtures/okf-minimal", "alpha", "--json"},
			code: 0,
		},
		{
			name: "daemon-status",
			args: []string{"daemon", "status", "--json"},
			env:  []string{"LLM_WIKI_STATE_DIR=" + filepath.Join(repo, "testdata", "runtime-state")},
			code: 0,
		},
		{
			name: "daemon-doctor",
			args: []string{"daemon", "doctor", "--json"},
			env:  []string{"LLM_WIKI_STATE_DIR=" + filepath.Join(repo, "testdata", "runtime-state")},
			code: 0,
		},
		{
			name: "daemon-start",
			args: []string{"daemon", "start", "--json"},
			env:  []string{"LLM_WIKI_STATE_DIR=" + filepath.Join(repo, "testdata", "runtime-state")},
			code: 2,
		},
		{
			name: "daemon-stop",
			args: []string{"daemon", "stop", "--json"},
			env:  []string{"LLM_WIKI_STATE_DIR=" + filepath.Join(repo, "testdata", "runtime-state")},
			code: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bin, tc.args...)
			cmd.Dir = repo
			cmd.Env = append(os.Environ(), tc.env...)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if code := exitCode(err); code != tc.code {
				t.Fatalf("exit code = %d, want %d\nstderr:\n%s", code, tc.code, stderr.String())
			}
			got := normalizeJSON(t, stdout.Bytes(), repo)
			wantPath := filepath.Join(repo, "testdata", "snapshots", tc.name+".json")
			want, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
				t.Fatalf("snapshot mismatch for %s\ngot:\n%s\nwant:\n%s", tc.name, got, want)
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	return -1
}

func normalizeJSON(t *testing.T, data []byte, repo string) []byte {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	normalizeValue(value, filepath.Clean(repo))
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n')
}

func normalizeValue(value any, repo string) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if s, ok := child.(string); ok {
				v[key] = normalizeString(s, repo)
				continue
			}
			normalizeValue(child, repo)
		}
	case []any:
		for _, child := range v {
			normalizeValue(child, repo)
		}
	}
}

func normalizeString(value string, repo string) string {
	value = filepath.ToSlash(value)
	repo = filepath.ToSlash(repo)
	if strings.HasPrefix(value, repo) {
		return "$REPO" + strings.TrimPrefix(value, repo)
	}
	return value
}
