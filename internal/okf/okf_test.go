package okf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCountsConceptsAndExcludesReservedFiles(t *testing.T) {
	bundle, err := Scan("../../fixtures/okf-minimal")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.ConceptCount() != 1 {
		t.Fatalf("concept count = %d, want 1", bundle.ConceptCount())
	}
	if len(bundle.ReservedFiles) != 2 {
		t.Fatalf("reserved files = %v", bundle.ReservedFiles)
	}
}

func TestNestedConceptPathStability(t *testing.T) {
	bundle, err := Scan("../../fixtures/okf-nested")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Concepts) != 1 || bundle.Concepts[0].RelPath != "nested/beta.md" {
		t.Fatalf("concepts = %#v, want nested/beta.md", bundle.Concepts)
	}
}

func TestSafeWritePathRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../outside.md", "link/escape.md"} {
		if _, err := SafeWritePath(root, rel); err == nil {
			t.Fatalf("SafeWritePath(%q) succeeded, want rejection", rel)
		}
	}
	if _, err := SafeWritePath(root, "nested/file.md"); err != nil {
		t.Fatalf("safe nested path rejected: %v", err)
	}
}
