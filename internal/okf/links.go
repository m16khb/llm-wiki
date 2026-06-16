package okf

import (
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type BundleLink struct {
	Target string
	Raw    string
	Kind   string
}

var wikiLinkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)

func ExtractBundleLinks(sourceRel, body string) []BundleLink {
	links := markdownBundleLinks(sourceRel, body)
	for _, target := range wikiLinkTargets(body) {
		normalized := normalizeRootLink(target)
		if normalized == "" {
			continue
		}
		links = append(links, BundleLink{Target: normalized, Raw: target, Kind: "wikilink"})
	}
	return links
}

func markdownBundleLinks(sourceRel, body string) []BundleLink {
	source := []byte(body)
	doc := goldmark.DefaultParser().Parse(text.NewReader(source))
	links := []BundleLink{}
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := node.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		raw := strings.TrimSpace(string(link.Destination))
		target := normalizeMarkdownLink(sourceRel, raw)
		if target == "" {
			return ast.WalkContinue, nil
		}
		links = append(links, BundleLink{Target: target, Raw: raw, Kind: "markdown"})
		return ast.WalkContinue, nil
	})
	return links
}

func wikiLinkTargets(body string) []string {
	matches := wikiLinkRE.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimSpace(match[1]))
	}
	return out
}

func normalizeMarkdownLink(sourceRel, raw string) string {
	if raw == "" || strings.HasPrefix(raw, "#") || isExternalLink(raw) {
		return ""
	}
	withoutFragment := strings.SplitN(raw, "#", 2)[0]
	withoutQuery := strings.SplitN(withoutFragment, "?", 2)[0]
	if withoutQuery == "" {
		return ""
	}
	if strings.HasPrefix(withoutQuery, "/") {
		return normalizeRootLink(strings.TrimPrefix(withoutQuery, "/"))
	}
	base := path.Dir(filepath.ToSlash(sourceRel))
	if base == "." {
		base = ""
	}
	return normalizeRootLink(path.Join(base, withoutQuery))
}

func normalizeRootLink(target string) string {
	target = strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(target), "/"))
	if target == "" {
		return ""
	}
	clean := path.Clean(target)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return ""
	}
	if path.Ext(clean) == "" {
		clean += ".md"
	}
	return clean
}

func isExternalLink(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(raw, "//")
}
