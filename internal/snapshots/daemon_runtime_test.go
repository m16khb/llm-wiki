package snapshots

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/m16khb/llm-wiki/internal/validate"
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

func TestMCPCommandUsesPerConnectionVaultWithoutRestartingDaemon(t *testing.T) {
	repo := repoRoot(t)
	bin := buildCLI(t, repo)
	stateDir := shortTempDir(t)
	stateEnv := "LLM_WIKI_STATE_DIR=" + stateDir
	t.Cleanup(func() {
		_ = runCLI(t, repo, bin, []string{stateEnv}, "daemon", "stop", "--json")
	})

	startedWithoutVault := runDaemonStatus(t, repo, bin, []string{stateEnv, "LLM_WIKI_VAULT="}, "start")
	if !startedWithoutVault.Running || startedWithoutVault.PID == 0 {
		t.Fatalf("initial daemon start = %#v, want running daemon without vault env", startedWithoutVault)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sessionA := connectMCPCommand(t, ctx, repo, bin, stateEnv, filepath.Join(repo, "fixtures", "okf-minimal"))
	defer sessionA.Close()

	resultA, err := sessionA.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dtoA validate.Result
	decodeStructuredContent(t, resultA.StructuredContent, &dtoA)
	if !dtoA.OK || dtoA.ConceptCount != 1 {
		t.Fatalf("dto A = %#v, want validate to default to proxy A vault env", dtoA)
	}

	sessionB := connectMCPCommand(t, ctx, repo, bin, stateEnv, filepath.Join(repo, "fixtures", "querypack-graph"))
	defer sessionB.Close()
	resultB, err := sessionB.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dtoB validate.Result
	decodeStructuredContent(t, resultB.StructuredContent, &dtoB)
	if !dtoB.OK || dtoB.ConceptCount != 4 {
		t.Fatalf("dto B = %#v, want validate to default to proxy B vault env", dtoB)
	}

	status := runDaemonStatus(t, repo, bin, []string{stateEnv}, "status")
	if status.PID != startedWithoutVault.PID {
		t.Fatalf("daemon pid = %d, want same pid %d across vault env changes", status.PID, startedWithoutVault.PID)
	}
}

func TestDaemonReplaceDrainsExistingMCPSession(t *testing.T) {
	repo := repoRoot(t)
	bin := buildCLI(t, repo)
	stateDir := shortTempDir(t)
	stateEnv := "LLM_WIKI_STATE_DIR=" + stateDir
	t.Cleanup(func() {
		_ = runCLI(t, repo, bin, []string{stateEnv}, "daemon", "stop", "--json")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	sessionA := connectMCPCommand(t, ctx, repo, bin, stateEnv, filepath.Join(repo, "fixtures", "okf-minimal"))
	statusBefore := runDaemonStatus(t, repo, bin, []string{stateEnv}, "status")
	if !statusBefore.Running || statusBefore.PID == 0 {
		t.Fatalf("daemon status before replace = %#v, want running daemon", statusBefore)
	}

	replaced := runDaemonStatus(t, repo, bin, []string{stateEnv}, "replace")
	if !replaced.Running || replaced.PID == 0 || replaced.PID == statusBefore.PID {
		t.Fatalf("daemon replace = %#v, want new running pid different from %d", replaced, statusBefore.PID)
	}

	resultA, err := sessionA.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dtoA validate.Result
	decodeStructuredContent(t, resultA.StructuredContent, &dtoA)
	if !dtoA.OK || dtoA.ConceptCount != 1 {
		t.Fatalf("dto A after replace = %#v, want old drained session to keep serving", dtoA)
	}

	sessionB := connectMCPCommand(t, ctx, repo, bin, stateEnv, filepath.Join(repo, "fixtures", "querypack-graph"))
	defer sessionB.Close()
	resultB, err := sessionB.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dtoB validate.Result
	decodeStructuredContent(t, resultB.StructuredContent, &dtoB)
	if !dtoB.OK || dtoB.ConceptCount != 4 {
		t.Fatalf("dto B after replace = %#v, want new session to use new daemon", dtoB)
	}

	if err := sessionA.Close(); err != nil {
		t.Fatal(err)
	}
	waitForPIDExit(t, statusBefore.PID, 10*time.Second)
}

func connectMCPCommand(t *testing.T, ctx context.Context, repo, bin, stateEnv, vaultPath string) *mcpsdk.ClientSession {
	t.Helper()
	cmd := exec.Command(bin, "mcp")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), stateEnv, "LLM_WIKI_VAULT="+vaultPath)
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "llm-wiki-daemon-test"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func waitForPIDExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		proc, err := os.FindProcess(pid)
		if err != nil || proc.Signal(os.Signal(syscall.Signal(0))) != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %s", pid, timeout)
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

func decodeStructuredContent(t *testing.T, value any, out any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode structured content: %v\n%s", err, data)
	}
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
