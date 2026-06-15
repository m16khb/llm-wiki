package validate

import (
	"fmt"

	"github.com/m16khb/llm-wiki/internal/okf"
)

type Result struct {
	OK            bool     `json:"ok"`
	OKFVersion    string   `json:"okf_version"`
	BundleRoot    string   `json:"bundle_root"`
	ConceptCount  int      `json:"concept_count"`
	ReservedFiles []string `json:"reserved_files"`
	Errors        []string `json:"errors"`
	Warnings      []string `json:"warnings"`
}

func Bundle(root string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		OK:            true,
		OKFVersion:    okf.Version,
		BundleRoot:    bundle.Root,
		ConceptCount:  bundle.ConceptCount(),
		ReservedFiles: append([]string(nil), bundle.ReservedFiles...),
		Errors:        []string{},
		Warnings:      []string{},
	}
	for _, concept := range bundle.Concepts {
		if concept.Type == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing required frontmatter field type", concept.RelPath))
		}
	}
	if len(result.Errors) > 0 {
		result.OK = false
	}
	return result, nil
}
