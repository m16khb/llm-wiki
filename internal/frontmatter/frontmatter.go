package frontmatter

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Markdown struct {
	node   *yaml.Node
	body   []byte
	hasYML bool
}

func ParseMarkdown(input []byte) (*Markdown, error) {
	text := string(input)
	if !strings.HasPrefix(text, "---\n") {
		return &Markdown{node: emptyMapNode(), body: input}, nil
	}
	end := strings.Index(text[len("---\n"):], "\n---")
	if end < 0 {
		return nil, fmt.Errorf("frontmatter start marker without end marker")
	}
	yml := text[len("---\n") : len("---\n")+end]
	rest := text[len("---\n")+end+len("\n---"):]
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}
	var doc yaml.Node
	if strings.TrimSpace(yml) == "" {
		doc.Content = []*yaml.Node{emptyMapNode()}
	} else if err := yaml.Unmarshal([]byte(yml), &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML mapping")
	}
	return &Markdown{node: doc.Content[0], body: []byte(rest), hasYML: true}, nil
}

func (m *Markdown) HasFrontmatter() bool {
	return m != nil && m.hasYML
}

func (m *Markdown) Body() []byte {
	if m == nil {
		return nil
	}
	return append([]byte(nil), m.body...)
}

func (m *Markdown) GetString(key string) string {
	if m == nil || m.node == nil {
		return ""
	}
	for i := 0; i+1 < len(m.node.Content); i += 2 {
		if m.node.Content[i].Value == key && m.node.Content[i+1].Kind == yaml.ScalarNode {
			return m.node.Content[i+1].Value
		}
	}
	return ""
}

func (m *Markdown) SetString(key, value string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("key is required")
	}
	if m.node == nil {
		m.node = emptyMapNode()
	}
	for i := 0; i+1 < len(m.node.Content); i += 2 {
		if m.node.Content[i].Value == key {
			m.node.Content[i+1] = scalar(value)
			m.hasYML = true
			return nil
		}
	}
	m.node.Content = append(m.node.Content, scalar(key), scalar(value))
	m.hasYML = true
	return nil
}

func (m *Markdown) Markdown() ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("nil markdown")
	}
	if !m.hasYML {
		return append([]byte(nil), m.body...), nil
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m.node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.Write(m.body)
	return buf.Bytes(), nil
}

func emptyMapNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}
