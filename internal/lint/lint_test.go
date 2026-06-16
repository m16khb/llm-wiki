package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintWarnsForBrokenLinksButDoesNotFailValidation(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-broken-link", false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("lint OK = false, errors = %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected broken link warning")
	}
}

func TestLintWarnsForBrokenMarkdownLinksOnly(t *testing.T) {
	root := t.TempDir()
	writeConcept(t, root, "alpha.md", "Alpha", strings.Join([]string{
		"[Beta](/nested/beta.md)",
		"[Missing](missing.md)",
		"[External](https://example.com/doc.md)",
		"[Email](mailto:test@example.com)",
		"[Section](#details)",
	}, "\n"))
	writeConcept(t, root, "nested/beta.md", "Beta", "# Beta\n")

	result, err := Bundle(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("lint OK = false, errors = %v", result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one broken local Markdown link warning", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "missing.md") {
		t.Fatalf("warning = %q, want missing.md", result.Warnings[0])
	}
}

func writeConcept(t *testing.T, root, rel, title, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntype: concept\ntitle: " + title + "\n---\n\n" + body
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
