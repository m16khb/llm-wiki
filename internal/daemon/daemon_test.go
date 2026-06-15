package daemon

import (
	"os"
	"path/filepath"
	"testing"
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

func abs(t *testing.T, elem ...string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(elem...))
	if err != nil {
		t.Fatal(err)
	}
	return path
}
