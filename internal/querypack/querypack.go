package querypack

import (
	"sort"
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

const maxContexts = 8

type Context struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type Result struct {
	OK          bool      `json:"ok"`
	BundleRoot  string    `json:"bundle_root"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer,omitempty"`
	ContextOnly bool      `json:"context_only"`
	Contexts    []Context `json:"contexts"`
}

func Build(root, question string) (Result, error) {
	bundle, err := okf.Scan(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{OK: true, BundleRoot: bundle.Root, Question: question, ContextOnly: true, Contexts: []Context{}}
	if strings.TrimSpace(question) == "" {
		for _, concept := range bundle.Concepts {
			result.Contexts = append(result.Contexts, contextFor(concept, nil))
			if len(result.Contexts) >= maxContexts {
				break
			}
		}
		return result, nil
	}
	result.Contexts = selectContexts(bundle, question)
	return result, nil
}

type scoredConcept struct {
	concept okf.Concept
	score   int
}

func selectContexts(bundle *okf.Bundle, question string) []Context {
	tokens := queryTokens(question)
	if len(tokens) == 0 {
		return []Context{}
	}
	q := strings.ToLower(strings.TrimSpace(question))
	scored := []scoredConcept{}
	for _, concept := range bundle.Concepts {
		score := conceptScore(concept, q, tokens)
		if score == 0 {
			continue
		}
		scored = append(scored, scoredConcept{concept: concept, score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].concept.RelPath < scored[j].concept.RelPath
	})

	contexts := []Context{}
	seen := map[string]bool{}
	for _, candidate := range scored {
		contexts = append(contexts, contextFor(candidate.concept, tokens))
		seen[candidate.concept.RelPath] = true
		if len(contexts) >= maxContexts {
			return contexts
		}
	}

	byPath, outbound, inbound := graphMaps(bundle)
	for _, seed := range scored {
		for _, neighbor := range append(outbound[seed.concept.RelPath], inbound[seed.concept.RelPath]...) {
			concept, ok := byPath[neighbor]
			if !ok || seen[neighbor] {
				continue
			}
			contexts = append(contexts, contextFor(concept, nil))
			seen[neighbor] = true
			if len(contexts) >= maxContexts {
				return contexts
			}
		}
	}
	return contexts
}
