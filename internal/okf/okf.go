package okf

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m16khb/llm-wiki/internal/frontmatter"
)

const Version = "0.1"

var ReservedFiles = []string{"index.md", "log.md"}

type Concept struct {
	RelPath string
	AbsPath string
	Type    string
	Title   string
	Body    string
}

type Bundle struct {
	Root          string
	Concepts      []Concept
	ReservedFiles []string
}

func Scan(root string) (*Bundle, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	bundle := &Bundle{Root: realRoot, ReservedFiles: append([]string(nil), ReservedFiles...)}
	err = filepath.WalkDir(realRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && path != realRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(realRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if isReserved(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		md, err := frontmatter.ParseMarkdown(data)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		bundle.Concepts = append(bundle.Concepts, Concept{
			RelPath: rel,
			AbsPath: path,
			Type:    md.GetString("type"),
			Title:   titleOrStem(md.GetString("title"), rel),
			Body:    string(md.Body()),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(bundle.Concepts, func(i, j int) bool { return bundle.Concepts[i].RelPath < bundle.Concepts[j].RelPath })
	return bundle, nil
}

func (b *Bundle) ConceptCount() int {
	if b == nil {
		return 0
	}
	return len(b.Concepts)
}

func SafeWritePath(root, rel string) (string, error) {
	if filepath.IsAbs(rel) || strings.Contains(rel, "\x00") {
		return "", fmt.Errorf("unsafe path %q", rel)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("path escapes bundle root: %s", rel)
	}
	target := filepath.Join(realRoot, clean)
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	prefix := realRoot + string(filepath.Separator)
	if realParent != realRoot && !strings.HasPrefix(realParent, prefix) {
		return "", fmt.Errorf("path escapes bundle root through symlink: %s", rel)
	}
	return target, nil
}

func isReserved(rel string) bool {
	for _, reserved := range ReservedFiles {
		if rel == reserved {
			return true
		}
	}
	return false
}

func titleOrStem(title, rel string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	return strings.ReplaceAll(base, "-", " ")
}
