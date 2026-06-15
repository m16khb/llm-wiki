package querypack

import "testing"

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
