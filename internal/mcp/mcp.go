package mcp

import (
	"context"
	"io"

	"github.com/m16khb/llm-wiki/internal/graph"
	"github.com/m16khb/llm-wiki/internal/index"
	"github.com/m16khb/llm-wiki/internal/lint"
	"github.com/m16khb/llm-wiki/internal/querypack"
	"github.com/m16khb/llm-wiki/internal/validate"
	"github.com/m16khb/llm-wiki/internal/vault"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

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
			"required": []string{"question"},
			"properties": map[string]any{
				"path":     map[string]any{"type": "string"},
				"question": map[string]any{"type": "string"},
			},
		}},
	}
}

type pathArgs struct {
	Path string `json:"path,omitempty" jsonschema:"Path to the OKF bundle root. Defaults to LLM_WIKI_VAULT when omitted."`
}

type lintArgs struct {
	Path string `json:"path,omitempty" jsonschema:"Path to the OKF bundle root. Defaults to LLM_WIKI_VAULT when omitted."`
	Fix  bool   `json:"fix,omitempty" jsonschema:"Apply safe lint fixes such as index generation."`
}

type queryPackArgs struct {
	Path     string `json:"path,omitempty" jsonschema:"Path to the OKF bundle root. Defaults to LLM_WIKI_VAULT when omitted."`
	Question string `json:"question" jsonschema:"Question used only for bounded context retrieval."`
}

func NewServer() *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "llm-wiki", Version: "0.1.0"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "llm_wiki_validate",
		Description: "Validate an OKF bundle and return the same DTO as `llm-wiki validate --json`.",
	}, validateTool)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "llm_wiki_lint",
		Description: "Lint an OKF bundle for quality warnings; broken links are warnings, not validation errors.",
	}, lintTool)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "llm_wiki_index",
		Description: "Write a deterministic `index.md` for an OKF bundle.",
	}, indexTool)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "llm_wiki_graph",
		Description: "Return deterministic concept nodes and wiki-link edges for an OKF bundle.",
	}, graphTool)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "llm_wiki_query_pack",
		Description: "Return bounded context for a question and never synthesize an answer.",
	}, queryPackTool)
	return server
}

func RunStdio(ctx context.Context) error {
	return NewServer().Run(ctx, &mcpsdk.StdioTransport{})
}

func RunStream(ctx context.Context, rwc io.ReadWriteCloser) error {
	return NewServer().Run(ctx, NewStreamTransport(rwc))
}

func validateTool(_ context.Context, _ *mcpsdk.CallToolRequest, args pathArgs) (*mcpsdk.CallToolResult, validate.Result, error) {
	root, err := vault.Resolve(args.Path)
	if err != nil {
		return nil, validate.Result{}, err
	}
	result, err := validate.Bundle(root)
	return nil, result, err
}

func lintTool(_ context.Context, _ *mcpsdk.CallToolRequest, args lintArgs) (*mcpsdk.CallToolResult, validate.Result, error) {
	root, err := vault.Resolve(args.Path)
	if err != nil {
		return nil, validate.Result{}, err
	}
	result, err := lint.Bundle(root, args.Fix)
	return nil, result, err
}

func indexTool(_ context.Context, _ *mcpsdk.CallToolRequest, args pathArgs) (*mcpsdk.CallToolResult, index.Result, error) {
	root, err := vault.Resolve(args.Path)
	if err != nil {
		return nil, index.Result{}, err
	}
	result, err := index.Write(root)
	return nil, result, err
}

func graphTool(_ context.Context, _ *mcpsdk.CallToolRequest, args pathArgs) (*mcpsdk.CallToolResult, graph.Result, error) {
	root, err := vault.Resolve(args.Path)
	if err != nil {
		return nil, graph.Result{}, err
	}
	result, err := graph.Build(root)
	return nil, result, err
}

func queryPackTool(_ context.Context, _ *mcpsdk.CallToolRequest, args queryPackArgs) (*mcpsdk.CallToolResult, querypack.Result, error) {
	root, err := vault.Resolve(args.Path)
	if err != nil {
		return nil, querypack.Result{}, err
	}
	result, err := querypack.Build(root, args.Question)
	return nil, result, err
}

func pathSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{"path": map[string]any{"type": "string"}},
	}
}
