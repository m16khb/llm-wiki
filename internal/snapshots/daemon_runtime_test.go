package snapshots

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type daemonStatusDTO struct {
	OK          bool   `json:"ok"`
	Action      string `json:"action"`
	Implemented bool   `json:"implemented"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid,omitempty"`
	StateDir    string `json:"state_dir"`
	SocketPath  string `json:"socket_path"`
	PIDPath     string `json:"pid_path"`
	LockPath    string `json:"lock_path"`
	Message     string `json:"message"`
}

func TestDaemonLifecycleCLIStartsAndStopsBackgroundServer(t *testing.T) {
	repo := repoRoot(t)
	bin := buildCLI(t, repo)
	stateDir := shortTempDir(t)
	env := []string{"LLM_WIKI_STATE_DIR=" + stateDir}
	t.Cleanup(func() {
		_ = runCLI(t, repo, bin, env, "daemon", "stop", "--json")
	})

	started := runDaemonStatus(t, repo, bin, env, "start")
	if !started.OK || !started.Implemented || !started.Running || started.PID == 0 {
		t.Fatalf("daemon start = %#v, want implemented running daemon with pid", started)
	}
	if _, err := os.Stat(started.SocketPath); err != nil {
		t.Fatalf("socket missing after start: %v", err)
	}

	status := runDaemonStatus(t, repo, bin, env, "status")
	if !status.OK || !status.Implemented || !status.Running || status.PID != started.PID {
		t.Fatalf("daemon status = %#v, want same running daemon pid %d", status, started.PID)
	}

	stopped := runDaemonStatus(t, repo, bin, env, "stop")
	if !stopped.OK || !stopped.Implemented || stopped.Running {
		t.Fatalf("daemon stop = %#v, want implemented stopped daemon", stopped)
	}
	if _, err := os.Stat(stopped.SocketPath); !os.IsNotExist(err) {
		t.Fatalf("socket still present after stop: %v", err)
	}
}

func TestMCPCommandAutoStartsDaemonAndProxiesTools(t *testing.T) {
	repo := repoRoot(t)
	bin := buildCLI(t, repo)
	stateDir := shortTempDir(t)
	t.Cleanup(func() {
		_ = runCLI(t, repo, bin, []string{"LLM_WIKI_STATE_DIR=" + stateDir}, "daemon", "stop", "--json")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.Command(bin, "mcp")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "LLM_WIKI_STATE_DIR="+stateDir)
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "llm-wiki-daemon-test"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	if !slices.Contains(names, "llm_wiki_validate") || !slices.Contains(names, "llm_wiki_query_pack") {
		t.Fatalf("tools = %v, want llm_wiki_validate and llm_wiki_query_pack", names)
	}

	status := runDaemonStatus(t, repo, bin, []string{"LLM_WIKI_STATE_DIR=" + stateDir}, "status")
	if !status.Running {
		t.Fatalf("daemon status after mcp = %#v, want running daemon", status)
	}
}

func buildCLI(t *testing.T, repo string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "llm-wiki")
	build := exec.Command("go", "build", "-o", bin, "./cmd/llm-wiki")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return bin
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "lw-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func runDaemonStatus(t *testing.T, repo, bin string, env []string, action string) daemonStatusDTO {
	t.Helper()
	out := runCLI(t, repo, bin, env, "daemon", action, "--json")
	var status daemonStatusDTO
	if err := json.Unmarshal(out, &status); err != nil {
		t.Fatalf("decode daemon %s: %v\n%s", action, err, out)
	}
	return status
}

func runCLI(t *testing.T, repo, bin string, env []string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", bin, args, err, out)
	}
	return out
}
