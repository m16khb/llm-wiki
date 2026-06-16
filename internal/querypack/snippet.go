package querypack

import (
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

func contextFor(concept okf.Concept, tokens []string) Context {
	return Context{Path: concept.RelPath, Title: concept.Title, Snippet: snippetFor(concept.Body, tokens, 500)}
}

func snippetFor(body string, tokens []string, max int) string {
	body = strings.TrimSpace(body)
	if len([]rune(body)) <= max {
		return body
	}
	if len(tokens) == 0 {
		return snippet(body, max)
	}
	lowerTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		lowerTokens = append(lowerTokens, strings.ToLower(token))
	}
	for _, paragraph := range paragraphs(body) {
		lower := strings.ToLower(paragraph)
		for _, token := range lowerTokens {
			if strings.Contains(lower, token) {
				return snippet(paragraph, max)
			}
		}
	}
	return snippet(body, max)
}

func paragraphs(body string) []string {
	parts := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func snippet(body string, max int) string {
	body = strings.TrimSpace(body)
	runes := []rune(body)
	if len(runes) <= max {
		return body
	}
	suffix := "...[truncated]"
	if max <= len([]rune(suffix)) {
		return string(runes[:max])
	}
	return string(runes[:max-len([]rune(suffix))]) + suffix
}
