package frontmatter

import (
	"strings"
	"testing"
)

func TestParseAndWritePreservesUnknownFields(t *testing.T) {
	doc := strings.Join([]string{
		"---",
		"type: concept",
		"title: Alpha",
		"custom: keep-me",
		"tags:",
		"  - okf",
		"---",
		"",
		"# Alpha",
	}, "\n")

	parsed, err := ParseMarkdown([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.GetString("type"); got != "concept" {
		t.Fatalf("type = %q, want concept", got)
	}
	if err := parsed.SetString("title", "Renamed"); err != nil {
		t.Fatal(err)
	}
	out, err := parsed.Markdown()
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{"custom: keep-me", "- okf", "title: Renamed", "# Alpha"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestParseMarkdownWithoutFrontmatter(t *testing.T) {
	parsed, err := ParseMarkdown([]byte("# Plain\n"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.HasFrontmatter() {
		t.Fatal("plain markdown unexpectedly had frontmatter")
	}
}
