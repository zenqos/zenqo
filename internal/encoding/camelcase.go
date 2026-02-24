package encoding

import (
	"strings"
	"unicode"
)

// ToCamelCase converts a Go PascalCase identifier to camelCase.
//
//	ID          → id
//	Name        → name
//	CreatedAt   → createdAt
//	UserID      → userId
//	HTMLContent → htmlContent
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var words [][]rune
	var cur []rune

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prevLower := unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || nextLower {
				words = append(words, cur)
				cur = []rune{r}
				continue
			}
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		words = append(words, cur)
	}

	var b strings.Builder
	for i, w := range words {
		lower := strings.ToLower(string(w))
		if i == 0 {
			b.WriteString(lower)
		} else {
			b.WriteString(strings.ToUpper(lower[:1]) + lower[1:])
		}
	}
	return b.String()
}
