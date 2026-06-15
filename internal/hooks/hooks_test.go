package hooks

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAppendEventRedactsAndCapsPayloadWithConcurrentWriters(t *testing.T) {
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := AppendEvent(root, Event{
				Event:   "PostToolUse",
				Host:    "codex",
				Message: strings.Repeat("x", 4096) + " token=secret-value",
			})
			if err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	f, err := os.Open(filepath.Join(root, ".llm-wiki", "hooks.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
		line := scanner.Text()
		if strings.Contains(line, "secret-value") {
			t.Fatalf("line contains unredacted secret: %s", line)
		}
		if len(line) > 2300 {
			t.Fatalf("line too large: %d", len(line))
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 50 {
		t.Fatalf("lines = %d, want 50", count)
	}
}

func TestHostOutputShapes(t *testing.T) {
	if _, ok := OutputForHost("claude", "PreToolUse", "deny")["hookSpecificOutput"]; !ok {
		t.Fatal("claude deny output missing hookSpecificOutput")
	}
	if got := OutputForHost("codex", "Stop", "block")["continue"]; got != true {
		t.Fatalf("codex stop block continue = %v", got)
	}
	if len(OutputForHost("reasonix", "PostToolUse", "noop")) != 0 {
		t.Fatal("reasonix noop output should be empty")
	}
}
