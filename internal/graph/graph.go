package graph

import (
	"regexp"
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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

var linkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)

func Build(root string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{OK: true, Root: bundle.Root, Nodes: []Node{}, Edges: []Edge{}}
	for _, concept := range bundle.Concepts {
		_ = parser.NewParser().Parse(text.NewReader([]byte(concept.Body)))
		result.Nodes = append(result.Nodes, Node{ID: concept.RelPath, Title: concept.Title, Type: concept.Type})
		for _, target := range extractLinks(concept.Body) {
			result.Edges = append(result.Edges, Edge{From: concept.RelPath, To: normalizeTarget(target), Type: "wiki_link"})
		}
	}
	return result, nil
}

func extractLinks(body string) []string {
	matches := linkRE.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimSpace(match[1]))
	}
	return out
}

func normalizeTarget(target string) string {
	if strings.HasSuffix(target, ".md") {
		return target
	}
	return target + ".md"
}
