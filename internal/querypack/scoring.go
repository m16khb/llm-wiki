package querypack

import (
	"path/filepath"
	"strings"

	"github.com/m16khb/llm-wiki/internal/okf"
)

func conceptScore(concept okf.Concept, question string, tokens []string) int {
	pathText := strings.ToLower(strings.TrimSuffix(concept.RelPath, filepath.Ext(concept.RelPath)))
	titleText := strings.ToLower(concept.Title)
	bodyText := strings.ToLower(concept.Body)

	score := 0
	if question == concept.RelPath || question == pathText || question == titleText {
		score += 1000
	}
	if containsToken(pathText, tokens) {
		score += 800
	}
	if containsPhrase(titleText, question) {
		score += 600
	}
	score += 120 * countTokenHits(titleText, tokens)
	if containsPhrase(bodyText, question) {
		score += 80
	}
	score += 20 * countTokenHits(bodyText, tokens)
	return score
}

func containsPhrase(text, phrase string) bool {
	phrase = strings.TrimSpace(phrase)
	return phrase != "" && strings.Contains(text, phrase)
}

func containsToken(text string, tokens []string) bool {
	return countTokenHits(text, tokens) > 0
}

func countTokenHits(text string, tokens []string) int {
	count := 0
	for _, token := range tokens {
		if strings.Contains(text, token) {
			count++
		}
	}
	return count
}
