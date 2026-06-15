package validate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/m16khb/llm-wiki/internal/frontmatter"
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
		if !concept.ValidUTF8 {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: file is not valid UTF-8", concept.RelPath))
			continue
		}
		if !concept.HasFrontmatter {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing YAML frontmatter", concept.RelPath))
			continue
		}
		if strings.TrimSpace(concept.Type) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing required frontmatter field type", concept.RelPath))
		}
	}
	reservedErrors, reservedWarnings, err := validateReservedFiles(bundle.Root)
	if err != nil {
		return Result{}, err
	}
	result.Errors = append(result.Errors, reservedErrors...)
	result.Warnings = append(result.Warnings, reservedWarnings...)
	if len(result.Errors) > 0 {
		result.OK = false
	}
	return result, nil
}

var logDateHeadingRE = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

func validateReservedFiles(root string) ([]string, []string, error) {
	errors := []string{}
	warnings := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if name != "index.md" && name != "log.md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			errors = append(errors, fmt.Sprintf("%s: file is not valid UTF-8", rel))
			return nil
		}
		md, err := frontmatter.ParseMarkdown(data)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", rel, err))
			return nil
		}
		switch name {
		case "index.md":
			indexErrors, indexWarnings := validateIndexFile(rel, md)
			errors = append(errors, indexErrors...)
			warnings = append(warnings, indexWarnings...)
		case "log.md":
			errors = append(errors, validateLogFile(rel, md, string(md.Body()))...)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return errors, warnings, nil
}

func validateIndexFile(rel string, md *frontmatter.Markdown) ([]string, []string) {
	if !md.HasFrontmatter() {
		return nil, nil
	}
	if rel != "index.md" {
		return []string{fmt.Sprintf("%s: index files must not contain frontmatter", rel)}, nil
	}
	keys := md.Keys()
	if len(keys) == 1 && keys[0] == "okf_version" {
		if version := md.GetString("okf_version"); version != "" && version != okf.Version {
			return nil, []string{fmt.Sprintf("%s: unsupported okf_version %q; attempting best-effort consumption", rel, version)}
		}
		return nil, nil
	}
	if slices.Contains(keys, "okf_version") && len(keys) == 1 {
		return nil, nil
	}
	return []string{fmt.Sprintf("%s: root index frontmatter may only declare okf_version", rel)}, nil
}

func validateLogFile(rel string, md *frontmatter.Markdown, body string) []string {
	errors := []string{}
	if md.HasFrontmatter() {
		errors = append(errors, fmt.Sprintf("%s: log files must not contain frontmatter", rel))
	}
	for _, match := range logDateHeadingRE.FindAllStringSubmatch(body, -1) {
		if !isISODate(match[1]) {
			errors = append(errors, fmt.Sprintf("%s: log date heading must use YYYY-MM-DD: %s", rel, match[1]))
		}
	}
	return errors
}

func isISODate(value string) bool {
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(value) {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
