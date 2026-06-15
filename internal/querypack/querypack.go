package querypack

import (
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

type Context struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type Result struct {
	OK          bool      `json:"ok"`
	BundleRoot  string    `json:"bundle_root"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer,omitempty"`
	ContextOnly bool      `json:"context_only"`
	Contexts    []Context `json:"contexts"`
}

func Build(root, question string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	q := strings.ToLower(question)
	result := Result{OK: true, BundleRoot: bundle.Root, Question: question, ContextOnly: true, Contexts: []Context{}}
	for _, concept := range bundle.Concepts {
		haystack := strings.ToLower(concept.RelPath + "\n" + concept.Title + "\n" + concept.Body)
		if q == "" || strings.Contains(haystack, q) {
			result.Contexts = append(result.Contexts, Context{Path: concept.RelPath, Title: concept.Title, Snippet: snippet(concept.Body, 500)})
		}
		if len(result.Contexts) >= 8 {
			break
		}
	}
	return result, nil
}

func snippet(body string, max int) string {
	body = strings.TrimSpace(body)
	if len(body) <= max {
		return body
	}
	return body[:max] + "...[truncated]"
}
