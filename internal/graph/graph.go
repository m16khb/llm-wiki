package graph

import (
	"github.com/m16khb/llm-wiki/internal/okf"
)

type Node struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type Result struct {
	OK    bool   `json:"ok"`
	Root  string `json:"bundle_root"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func Build(root string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{OK: true, Root: bundle.Root, Nodes: []Node{}, Edges: []Edge{}}
	for _, concept := range bundle.Concepts {
		result.Nodes = append(result.Nodes, Node{ID: concept.RelPath, Title: concept.Title, Type: concept.Type})
		for _, link := range okf.ExtractBundleLinks(concept.RelPath, concept.Body) {
			result.Edges = append(result.Edges, Edge{From: concept.RelPath, To: link.Target, Type: "wiki_link"})
		}
	}
	return result, nil
}
