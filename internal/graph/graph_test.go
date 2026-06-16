package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGraphIncludesWikiLinkEdges(t *testing.T) {
	result, err := Build("../../fixtures/okf-minimal")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 {
		t.Fatalf("nodes = %v, want one concept", result.Nodes)
	}
	if len(result.Edges) != 1 || result.Edges[0].To != "nested/beta.md" {
		t.Fatalf("edges = %#v, want nested/beta.md edge", result.Edges)
	}
}

func TestGraphIncludesMarkdownAndWikiLinkEdges(t *testing.T) {
	root := t.TempDir()
	writeConcept(t, root, "alpha.md", "Alpha", "[Beta](/nested/beta.md)\n[Gamma](section/gamma.md)\n")
	writeConcept(t, root, "section/gamma.md", "Gamma", "[Beta](../nested/beta.md)\n[[nested/beta]]\n")
	writeConcept(t, root, "nested/beta.md", "Beta", "# Beta\n")

	result, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}

	edges := map[string]bool{}
	for _, edge := range result.Edges {
		edges[edge.From+"->"+edge.To] = true
	}
	for _, want := range []string{
		"alpha.md->nested/beta.md",
		"alpha.md->section/gamma.md",
		"section/gamma.md->nested/beta.md",
	} {
		if !edges[want] {
			t.Fatalf("edges = %#v, missing %s", result.Edges, want)
		}
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
