package lint

import "testing"

func TestLintWarnsForBrokenLinksButDoesNotFailValidation(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-broken-link", false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("lint OK = false, errors = %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected broken link warning")
	}
}
