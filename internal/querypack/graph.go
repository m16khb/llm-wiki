package querypack

import (
	"sort"

	"github.com/m16khb/llm-wiki/internal/okf"
)

func graphMaps(bundle *okf.Bundle) (map[string]okf.Concept, map[string][]string, map[string][]string) {
	byPath := map[string]okf.Concept{}
	for _, concept := range bundle.Concepts {
		byPath[concept.RelPath] = concept
	}
	outbound := map[string][]string{}
	inbound := map[string][]string{}
	for _, concept := range bundle.Concepts {
		seenLinks := map[string]bool{}
		for _, link := range okf.ExtractBundleLinks(concept.RelPath, concept.Body) {
			if _, ok := byPath[link.Target]; !ok || seenLinks[link.Target] {
				continue
			}
			outbound[concept.RelPath] = append(outbound[concept.RelPath], link.Target)
			inbound[link.Target] = append(inbound[link.Target], concept.RelPath)
			seenLinks[link.Target] = true
		}
	}
	for path := range outbound {
		sort.Strings(outbound[path])
	}
	for path := range inbound {
		sort.Strings(inbound[path])
	}
	return byPath, outbound, inbound
}
