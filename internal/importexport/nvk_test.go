package importexport

import "testing"

func TestNVKDryRunDoesNotWriteAndReturnsPlan(t *testing.T) {
	result, err := NVK("import", "../../fixtures/okf-minimal", t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || result.Action != "import" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected planned files")
	}
}
