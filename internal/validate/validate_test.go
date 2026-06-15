package validate

import "testing"

func TestValidateJSONDTOForMinimalBundle(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-minimal")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("OK = false, errors = %v", result.Errors)
	}
	if result.OKFVersion != "0.1" {
		t.Fatalf("okf_version = %q", result.OKFVersion)
	}
	if result.ConceptCount != 1 {
		t.Fatalf("concept_count = %d, want 1", result.ConceptCount)
	}
}

func TestValidateReportsMissingType(t *testing.T) {
	result, err := Bundle("../../fixtures/okf-invalid-missing-type")
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("validation unexpectedly passed")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want one missing type error", result.Errors)
	}
}
