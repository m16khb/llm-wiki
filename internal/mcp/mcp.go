package mcp

import _ "github.com/modelcontextprotocol/go-sdk/mcp"

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

func Tools() []Tool {
	return []Tool{
		{Name: "llm_wiki_validate", Description: "Validate an OKF bundle and return the same DTO as the CLI validate --json command.", InputSchema: pathSchema()},
		{Name: "llm_wiki_lint", Description: "Lint an OKF bundle for quality warnings without synthesizing answers.", InputSchema: pathSchema()},
		{Name: "llm_wiki_index", Description: "Write a deterministic index.md for an OKF bundle.", InputSchema: pathSchema()},
		{Name: "llm_wiki_graph", Description: "Return deterministic nodes and links for an OKF bundle.", InputSchema: pathSchema()},
		{Name: "llm_wiki_query_pack", Description: "Return bounded context for a question and never synthesize an answer.", InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"path", "question"},
			"properties": map[string]any{
				"path":     map[string]any{"type": "string"},
				"question": map[string]any{"type": "string"},
			},
		}},
	}
}

func pathSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"required":   []string{"path"},
		"properties": map[string]any{"path": map[string]any{"type": "string"}},
	}
}
