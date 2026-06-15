package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteIndexCreatesDeterministicConceptList(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.md"), []byte("---\ntype: concept\ntitle: Alpha\n---\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Write(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatal("index write failed")
	}
	b, err := os.ReadFile(filepath.Join(root, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "- [Alpha](alpha.md)") {
		t.Fatalf("index missing concept link:\n%s", string(b))
	}
}
