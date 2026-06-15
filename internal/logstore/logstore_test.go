package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAppendUsesLockForConcurrentWriters(t *testing.T) {
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := Append(root, "test", "message"); err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	b, err := os.ReadFile(filepath.Join(root, "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(b), "- op: test"); got != 50 {
		t.Fatalf("log entries = %d, want 50\n%s", got, string(b))
	}
}
