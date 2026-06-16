package mcp

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/m16khb/llm-wiki/internal/validate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerListsExpectedTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientSession, serverSession := connectTestServer(t, ctx)
	defer serverSession.Wait()
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"llm_wiki_validate", "llm_wiki_lint", "llm_wiki_graph", "llm_wiki_query_pack"} {
		if !slices.Contains(names, want) {
			t.Fatalf("tools = %v, want %s", names, want)
		}
	}
}

func TestServerValidateToolReturnsCLICompatibleDTO(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientSession, serverSession := connectTestServer(t, ctx)
	defer serverSession.Wait()
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{"path": "../../fixtures/okf-minimal"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dto validate.Result
	decodeStructured(t, result.StructuredContent, &dto)
	if !dto.OK || dto.OKFVersion != "0.1" || dto.ConceptCount != 1 {
		t.Fatalf("dto = %#v, want valid OKF v0.1 bundle", dto)
	}
}

func TestServerValidateToolDefaultsToConfiguredVault(t *testing.T) {
	t.Setenv("LLM_WIKI_VAULT", "../../fixtures/okf-minimal")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientSession, serverSession := connectTestServer(t, ctx)
	defer serverSession.Wait()
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_validate",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dto validate.Result
	decodeStructured(t, result.StructuredContent, &dto)
	if !dto.OK || dto.BundleRoot == "" || dto.ConceptCount != 1 {
		t.Fatalf("dto = %#v, want vault-backed valid OKF bundle", dto)
	}
}

func TestServerQueryPackToolReturnsContextOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientSession, serverSession := connectTestServer(t, ctx)
	defer serverSession.Wait()
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "llm_wiki_query_pack",
		Arguments: map[string]any{"path": "../../fixtures/okf-minimal", "question": "alpha"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dto struct {
		ContextOnly bool   `json:"context_only"`
		Answer      string `json:"answer,omitempty"`
		Contexts    []struct {
			Path string `json:"path"`
		} `json:"contexts"`
	}
	decodeStructured(t, result.StructuredContent, &dto)
	if !dto.ContextOnly || dto.Answer != "" || len(dto.Contexts) != 1 {
		t.Fatalf("dto = %#v, want context-only query pack", dto)
	}
}

func connectTestServer(t *testing.T, ctx context.Context) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := NewServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "llm-wiki-test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Wait()
		t.Fatal(err)
	}
	return clientSession, serverSession
}

func decodeStructured(t *testing.T, value any, target any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode structured content: %v; raw=%s", err, string(data))
	}
}
