package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/m16khb/llm-wiki/internal/index"
	"github.com/m16khb/llm-wiki/internal/okf"
	"github.com/m16khb/llm-wiki/internal/validate"
)

var wikiLinkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)

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
		for _, target := range wikiLinks(concept.Body) {
			if !linkExists(bundle.Root, target) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: broken wiki link %q", concept.RelPath, target))
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

func wikiLinks(body string) []string {
	matches := wikiLinkRE.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimSpace(match[1]))
	}
	return out
}

func linkExists(root, target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	candidates := []string{target}
	if filepath.Ext(target) == "" {
		candidates = append(candidates, target+".md")
	}
	for _, candidate := range candidates {
		path := filepath.Join(root, filepath.FromSlash(candidate))
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}
