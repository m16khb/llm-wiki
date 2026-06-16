package daemon

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/m16khb/llm-wiki/internal/vault"
)

func TestPathsPreferLLMWikiStateDir(t *testing.T) {
	t.Setenv("LLM_WIKI_STATE_DIR", filepath.Join("tmp", "state"))
	t.Setenv("XDG_STATE_HOME", filepath.Join("tmp", "xdg"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(abs(t, "tmp", "state"))
	if paths.StateDir != want {
		t.Fatalf("StateDir = %q, want %q", paths.StateDir, want)
	}
	if paths.SocketPath != filepath.Join(want, "daemon.sock") {
		t.Fatalf("SocketPath = %q", paths.SocketPath)
	}
	if paths.PIDPath != filepath.Join(want, "daemon.pid") {
		t.Fatalf("PIDPath = %q", paths.PIDPath)
	}
	if paths.LockPath != filepath.Join(want, "daemon.lock") {
		t.Fatalf("LockPath = %q", paths.LockPath)
	}
}

func TestPathsUseXDGStateHome(t *testing.T) {
	t.Setenv("LLM_WIKI_STATE_DIR", "")
	t.Setenv("XDG_STATE_HOME", filepath.Join("tmp", "xdg"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(abs(t, "tmp", "xdg"), "llm-wiki")
	if paths.StateDir != want {
		t.Fatalf("StateDir = %q, want %q", paths.StateDir, want)
	}
}

func TestStatusReportsImplementedAndNotRunning(t *testing.T) {
	t.Setenv("LLM_WIKI_STATE_DIR", filepath.Join("tmp", "state"))

	result, err := Status()
	if err != nil {
		t.Fatal(err)
	}

	if !result.OK {
		t.Fatal("Status OK = false, want true")
	}
	if result.Action != "status" {
		t.Fatalf("Action = %q, want status", result.Action)
	}
	if !result.Implemented {
		t.Fatal("Implemented = false, want true")
	}
	if result.Running {
		t.Fatal("Running = true, want false")
	}
	if result.StateDir == "" || result.SocketPath == "" || result.PIDPath == "" || result.LockPath == "" {
		t.Fatalf("paths must be populated: %+v", result)
	}
	if result.Message == "" {
		t.Fatal("Message is empty")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want empty", result.Warnings)
	}
}

func TestStopIsIdempotentWhenDaemonIsNotRunning(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv("LLM_WIKI_STATE_DIR", stateDir)

	result, err := Stop()
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatal("OK = false, want true")
	}
	if result.Action != "stop" {
		t.Fatalf("Action = %q, want stop", result.Action)
	}
	if !result.Implemented {
		t.Fatal("Implemented = false, want true")
	}
	if result.Running {
		t.Fatal("Running = true, want false")
	}
	for _, path := range []string{result.SocketPath, result.PIDPath, result.LockPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s exists or stat failed with unexpected error: %v", path, err)
		}
	}
}

func TestDaemonServerCommandDetachesFromParentSession(t *testing.T) {
	logFile, err := os.CreateTemp(t.TempDir(), "daemon-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	paths := Paths{StateDir: t.TempDir()}
	cmd := daemonServerCommand("/tmp/llm-wiki", paths, logFile)

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatalf("SysProcAttr = %#v, want Setsid=true so daemon survives parent shell exit", cmd.SysProcAttr)
	}
	if cmd.Stdin != nil || cmd.Stdout != logFile || cmd.Stderr != logFile {
		t.Fatalf("stdio not daemon-safe: stdin=%#v stdout=%#v stderr=%#v", cmd.Stdin, cmd.Stdout, cmd.Stderr)
	}
}

func TestCleanupSiblingDaemonsKeepsCurrentPIDAndStopsStaleSiblings(t *testing.T) {
	originalExecutable := currentExecutable
	originalFind := findSiblingDaemonPIDs
	originalStateDir := daemonStateDirForPID
	originalProcessAlive := processAlivePID
	originalSignal := signalPID
	originalKill := killPID
	t.Cleanup(func() {
		currentExecutable = originalExecutable
		findSiblingDaemonPIDs = originalFind
		daemonStateDirForPID = originalStateDir
		processAlivePID = originalProcessAlive
		signalPID = originalSignal
		killPID = originalKill
	})

	currentExecutable = func() (string, error) { return "/tmp/llm-wiki", nil }
	findSiblingDaemonPIDs = func(exe string) ([]int, error) {
		if exe != "/tmp/llm-wiki" {
			t.Fatalf("exe = %q", exe)
		}
		return []int{111, 222, 333, os.Getpid()}, nil
	}
	daemonStateDirForPID = func(pid int) (string, bool) {
		switch pid {
		case 111, 222:
			return "/tmp/state-a", true
		case 333:
			return "/tmp/state-b", true
		default:
			return "", false
		}
	}
	signaled := []int{}
	processAlivePID = func(pid int) bool { return false }
	signalPID = func(pid int, signal syscall.Signal) error {
		if signal != syscall.SIGTERM {
			t.Fatalf("signal = %v, want SIGTERM", signal)
		}
		signaled = append(signaled, pid)
		return nil
	}
	killPID = func(pid int) error {
		t.Fatalf("killPID(%d) called for already-dead fake process", pid)
		return nil
	}

	warnings := cleanupSiblingDaemons("/tmp/state-a", 222)
	if len(signaled) != 1 || signaled[0] != 111 {
		t.Fatalf("signaled = %v, want only stale pid 111", signaled)
	}
	if len(warnings) != 1 || warnings[0] != "stopped stale daemon pid 111" {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestDaemonVaultMatchesCurrentEnv(t *testing.T) {
	originalVaultForPID := daemonVaultForPID
	t.Cleanup(func() {
		daemonVaultForPID = originalVaultForPID
	})

	t.Setenv(vault.EnvVar, "/tmp/current-vault")
	daemonVaultForPID = func(pid int) (string, bool) {
		if pid != 123 {
			t.Fatalf("pid = %d, want 123", pid)
		}
		return "/tmp/current-vault", true
	}

	if !daemonVaultMatchesCurrentEnv(123) {
		t.Fatal("daemonVaultMatchesCurrentEnv = false, want true for matching vault")
	}
}

func TestDaemonVaultMatchesCurrentEnvDetectsMissingDaemonVault(t *testing.T) {
	originalVaultForPID := daemonVaultForPID
	t.Cleanup(func() {
		daemonVaultForPID = originalVaultForPID
	})

	t.Setenv(vault.EnvVar, "/tmp/current-vault")
	daemonVaultForPID = func(pid int) (string, bool) {
		return "", false
	}

	if daemonVaultMatchesCurrentEnv(123) {
		t.Fatal("daemonVaultMatchesCurrentEnv = true, want false when current proxy has a vault but daemon does not")
	}
}

func TestDaemonVaultMatchesCurrentEnvDetectsStaleDaemonVault(t *testing.T) {
	originalVaultForPID := daemonVaultForPID
	t.Cleanup(func() {
		daemonVaultForPID = originalVaultForPID
	})

	t.Setenv(vault.EnvVar, "/tmp/current-vault")
	daemonVaultForPID = func(pid int) (string, bool) {
		return "/tmp/old-vault", true
	}

	if daemonVaultMatchesCurrentEnv(123) {
		t.Fatal("daemonVaultMatchesCurrentEnv = true, want false for stale daemon vault")
	}
}

func TestParsePIDsIgnoresNonNumericLines(t *testing.T) {
	got := parsePIDs("123\nnot-a-pid\n456\n")
	want := []int{123, 456}
	if len(got) != len(want) {
		t.Fatalf("parsePIDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parsePIDs = %v, want %v", got, want)
		}
	}
}

func abs(t *testing.T, elem ...string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(elem...))
	if err != nil {
		t.Fatal(err)
	}
	return path
}
