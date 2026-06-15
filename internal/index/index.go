package index

import (
	"fmt"
	"os"
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

type Result struct {
	OK         bool   `json:"ok"`
	BundleRoot string `json:"bundle_root"`
	Path       string `json:"path"`
	Count      int    `json:"count"`
}

func Write(root string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	var b strings.Builder
	b.WriteString("# Index\n\n")
	for _, concept := range bundle.Concepts {
		b.WriteString(fmt.Sprintf("- [%s](%s)\n", concept.Title, concept.RelPath))
	}
	path, err := okf.SafeWritePath(bundle.Root, "index.md")
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return Result{}, err
	}
	return Result{OK: true, BundleRoot: bundle.Root, Path: path, Count: len(bundle.Concepts)}, nil
}
