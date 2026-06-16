package querypack

import (
	"strings"
	"unicode"
)

func queryTokens(question string) []string {
	rawTokens := splitTokens(question)
	seen := map[string]bool{}
	out := []string{}
	add := func(token string) {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" || stopword(token) || seen[token] {
			return
		}
		seen[token] = true
		out = append(out, token)
	}
	for _, raw := range rawTokens {
		lower := strings.ToLower(raw)
		add(lower)
		for _, ascii := range asciiSubtokens(lower) {
			add(ascii)
		}
		for _, cjk := range cjkSubtokens(lower) {
			if runeLen(cjk) < 3 || stopwordLike(cjk) {
				continue
			}
			for _, bigram := range bigrams(cjk) {
				add(bigram)
			}
		}
	}
	return out
}

func splitTokens(s string) []string {
	tokens := []string{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func asciiSubtokens(token string) []string {
	out := []string{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range token {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	if len(out) == 1 && out[0] == token {
		return nil
	}
	return out
}

func bigrams(token string) []string {
	runes := []rune(token)
	out := make([]string, 0, len(runes)-1)
	for i := 0; i < len(runes)-1; i++ {
		out = append(out, string(runes[i:i+2]))
	}
	return out
}

func cjkSubtokens(token string) []string {
	out := []string{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range token {
		if isCJK(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Hangul, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

func runeLen(s string) int {
	return len([]rune(s))
}

func stopword(token string) bool {
	switch token {
	case "what", "how", "does", "무엇", "무엇인가", "어떻게", "란", "은", "는", "이", "가", "을", "를":
		return true
	default:
		return false
	}
}

func stopwordLike(token string) bool {
	return stopword(token) || strings.Contains(token, "무엇") || strings.Contains(token, "어떻게")
}
