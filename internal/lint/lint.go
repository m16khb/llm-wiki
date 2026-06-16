package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/m16khb/llm-wiki/internal/index"
	"github.com/m16khb/llm-wiki/internal/okf"
	"github.com/m16khb/llm-wiki/internal/validate"
)

type Result = validate.Result

func Bundle(root string, fix bool) (Result, error) {
	result, err := validate.Bundle(root)
	if err != nil {
		return Result{}, err
	}
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	for _, concept := range bundle.Concepts {
		for _, link := range okf.ExtractBundleLinks(concept.RelPath, concept.Body) {
			if !linkExists(bundle.Root, link.Target) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: broken local link %q", concept.RelPath, link.Target))
			}
		}
	}
	if fix {
		if _, err := index.Write(root); err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func linkExists(root, target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	path := filepath.Join(root, filepath.FromSlash(target))
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
