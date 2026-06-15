package daemon

import (
	"errors"
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

func TestStatusReportsUnimplementedAndNotRunning(t *testing.T) {
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
	if result.Implemented {
		t.Fatal("Implemented = true, want false")
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

func TestStartAndStopReturnUnsupportedResults(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv("LLM_WIKI_STATE_DIR", stateDir)

	for _, tc := range []struct {
		name   string
		action string
		run    func() (Result, error)
	}{
		{name: "start", action: "start", run: Start},
		{name: "stop", action: "stop", run: Stop},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.run()
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("error = %v, want ErrUnsupported", err)
			}
			if result.OK {
				t.Fatal("OK = true, want false")
			}
			if result.Action != tc.action {
				t.Fatalf("Action = %q, want %q", result.Action, tc.action)
			}
			if result.Implemented {
				t.Fatal("Implemented = true, want false")
			}
			if result.Running {
				t.Fatal("Running = true, want false")
			}
			if len(result.Warnings) == 0 {
				t.Fatal("Warnings is empty, want unsupported warning")
			}
			for _, path := range []string{result.StateDir, result.SocketPath, result.PIDPath, result.LockPath} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("%s exists or stat failed with unexpected error: %v", path, err)
				}
			}
		})
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
