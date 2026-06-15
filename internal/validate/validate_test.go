package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateJSONDTOForMinimalBundle(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-minimal")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("OK = false, errors = %v", result.Errors)
	}
	if result.OKFVersion != "0.1" {
		t.Fatalf("okf_version = %q", result.OKFVersion)
	}
	if result.ConceptCount != 1 {
		t.Fatalf("concept_count = %d, want 1", result.ConceptCount)
	}
}

func TestValidateReportsMissingType(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-invalid-missing-type")
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("validation unexpectedly passed")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want one missing type error", result.Errors)
	}
}

func TestValidateReportsMissingFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plain.md"), []byte("# Plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Bundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("validation unexpectedly passed")
	}
	if !contains(result.Errors, "plain.md: missing YAML frontmatter") {
		t.Fatalf("errors = %v, want missing frontmatter error", result.Errors)
	}
}

func TestValidateRejectsInvalidUTF8Concept(t *testing.T) {
	root := t.TempDir()
	data := []byte{'-', '-', '-', '\n', 't', 'y', 'p', 'e', ':', ' ', 'c', 'o', 'n', 'c', 'e', 'p', 't', '\n', '-', '-', '-', '\n', 0xff}
	if err := os.WriteFile(filepath.Join(root, "bad.md"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Bundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("validation unexpectedly passed")
	}
	if !contains(result.Errors, "bad.md: file is not valid UTF-8") {
		t.Fatalf("errors = %v, want invalid UTF-8 error", result.Errors)
	}
}

func TestValidateReservedFilesAtAnyLevel(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"nested/concept.md": "---\ntype: concept\ntitle: Nested\n---\n# Nested\n",
		"nested/index.md":   "---\ntitle: Bad Index\n---\n# Index\n",
		"nested/log.md":     "# Directory Update Log\n\n## 2026/05/22\n* **Update**: Bad date.\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Bundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.ConceptCount != 1 {
		t.Fatalf("concept_count = %d, want reserved nested files excluded", result.ConceptCount)
	}
	for _, want := range []string{
		"nested/index.md: index files must not contain frontmatter",
		"nested/log.md: log date heading must use YYYY-MM-DD",
	} {
		if !contains(result.Errors, want) {
			t.Fatalf("errors = %v, want %q", result.Errors, want)
		}
	}
}

func TestValidateAllowsRootIndexOKFVersionFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("---\nokf_version: \"0.1\"\n---\n# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "concept.md"), []byte("---\ntype: concept\n---\n# Concept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Bundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("validation failed: %v", result.Errors)
	}
	if result.ConceptCount != 1 {
		t.Fatalf("concept_count = %d, want 1", result.ConceptCount)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
