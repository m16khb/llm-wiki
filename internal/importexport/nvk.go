package importexport

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

type Result struct {
	OK     bool     `json:"ok"`
	Action string   `json:"action"`
	Source string   `json:"source"`
	Dest   string   `json:"dest"`
	DryRun bool     `json:"dry_run"`
	Files  []string `json:"files"`
}

func NVK(action, source, dest string, dryRun bool) (Result, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return Result{}, err
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return Result{}, err
	}
	result := Result{OK: true, Action: action, Source: absSource, Dest: absDest, DryRun: dryRun, Files: []string{}}
	err = filepath.WalkDir(absSource, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(absSource, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "index.md" || rel == "log.md" {
			return nil
		}
		result.Files = append(result.Files, rel)
		if dryRun {
			return nil
		}
		target, err := okf.SafeWritePath(absDest, rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return Result{}, err
	}
	result.Action = strings.ToLower(action)
	return result, nil
}
