package querypack

import (
	"encoding/json"
	"testing"
)

func TestQueryPackReturnsBoundedContextWithoutAnswer(t *testing.T) {
	result, err := Build("../../fixtures/okf-minimal", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "" {
		t.Fatalf("answer = %q, want empty", result.Answer)
	}
	if len(result.Contexts) != 1 {
		t.Fatalf("contexts = %#v, want one", result.Contexts)
	}
	if len(result.Contexts[0].Snippet) > 500 {
		t.Fatalf("snippet too long: %d", len(result.Contexts[0].Snippet))
	}
}

func TestQueryPackFindsNaturalLanguageQuestionByToken(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "MCP란 무엇인가?")
	if err != nil {
		t.Fatal(err)
	}
	paths := contextPaths(result)
	if len(paths) < 2 || paths[0] != "mcp.md" || paths[1] != "agent-graph.md" {
		t.Fatalf("paths = %v, want mcp.md followed by linked agent-graph.md", paths)
	}
	if containsPath(paths, "missing-concept.md") {
		t.Fatalf("paths = %v, want broken graph link ignored", paths)
	}
}

func TestQueryPackFindsMixedScriptAcronymToken(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "MCP란")
	if err != nil {
		t.Fatal(err)
	}
	paths := contextPaths(result)
	if len(paths) == 0 || paths[0] != "mcp.md" {
		t.Fatalf("paths = %v, want mcp.md first", paths)
	}
}

func TestQueryPackRanksTitleAndPathBeforeBodyOnlyMatches(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "agent graph")
	if err != nil {
		t.Fatal(err)
	}
	paths := contextPaths(result)
	if len(paths) < 2 || paths[0] != "agent-graph.md" || paths[1] != "mcp.md" {
		t.Fatalf("paths = %v, want title/path match before body-only match", paths)
	}
}

func TestQueryPackFindsTokenOverlapWithoutExactPhrase(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "Stop hook continue decision")
	if err != nil {
		t.Fatal(err)
	}
	paths := contextPaths(result)
	if len(paths) == 0 || paths[0] != "stop-hook.md" {
		t.Fatalf("paths = %v, want stop-hook.md first", paths)
	}
}

func TestQueryPackNoMatchKeepsEmptyContextOnlyDTO(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "quantum banana")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || !result.ContextOnly || result.Answer != "" || len(result.Contexts) != 0 {
		t.Fatalf("result = %#v, want ok context-only empty result", result)
	}
}

func TestQueryPackEmptyQuestionReturnsFirstEightByPath(t *testing.T) {
	result, err := Build("../../fixtures/querypack-graph", "")
	if err != nil {
		t.Fatal(err)
	}
	paths := contextPaths(result)
	want := []string{"agent-graph.md", "mcp.md", "noise.md", "stop-hook.md"}
	if !equalStrings(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestQueryPackRepeatedCallsAreByteIdentical(t *testing.T) {
	first, err := Build("../../fixtures/querypack-graph", "MCP란 무엇인가?")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build("../../fixtures/querypack-graph", "MCP란 무엇인가?")
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("first = %s\nsecond = %s", firstJSON, secondJSON)
	}
}

func contextPaths(result Result) []string {
	paths := make([]string, 0, len(result.Contexts))
	for _, context := range result.Contexts {
		paths = append(paths, context.Path)
	}
	return paths
}

func containsPath(paths []string, path string) bool {
	for _, candidate := range paths {
		if candidate == path {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
