package graph

import "testing"

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
