package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// jsonToYAML converts a JSON-encoded document to block-style YAML.
//
// No external dependencies are required: encoding/json is used to parse the
// input, and the result is formatted using a small recursive emitter. The
// output is always stable — map keys are sorted alphabetically — which is
// important for diffing and caching.
func jsonToYAML(data []byte) ([]byte, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("jsonToYAML: %w", err)
	}
	var buf strings.Builder
	emitYAML(&buf, root, 0)
	return []byte(buf.String()), nil
}

// emitYAML writes v to buf as YAML at the given indentation depth.
func emitYAML(buf *strings.Builder, v any, depth int) {
	pad := strings.Repeat("  ", depth)

	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			child := val[k]
			buf.WriteString(pad)
			buf.WriteString(yamlKey(k))
			buf.WriteByte(':')
			switch child.(type) {
			case map[string]any, []any:
				buf.WriteByte('\n')
				emitYAML(buf, child, depth+1)
			default:
				buf.WriteByte(' ')
				emitYAMLScalar(buf, child)
			}
		}

	case []any:
		for _, item := range val {
			buf.WriteString(pad)
			buf.WriteString("- ")
			switch child := item.(type) {
			case map[string]any:
				emitYAMLMapInline(buf, child, depth+1)
			case []any:
				buf.WriteByte('\n')
				emitYAML(buf, child, depth+1)
			default:
				emitYAMLScalar(buf, item)
			}
		}

	default:
		emitYAMLScalar(buf, v)
	}
}

// emitYAMLMapInline writes a map where the first key follows "- " on the same line.
func emitYAMLMapInline(buf *strings.Builder, m map[string]any, depth int) {
	pad := strings.Repeat("  ", depth)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			buf.WriteString(pad)
		}
		buf.WriteString(yamlKey(k))
		buf.WriteByte(':')
		child := m[k]
		switch child.(type) {
		case map[string]any, []any:
			buf.WriteByte('\n')
			emitYAML(buf, child, depth+1)
		default:
			buf.WriteByte(' ')
			emitYAMLScalar(buf, child)
		}
	}
}

// emitYAMLScalar writes a leaf value followed by a newline.
func emitYAMLScalar(buf *strings.Builder, v any) {
	switch val := v.(type) {
	case string:
		buf.WriteString(yamlString(val))
	case float64:
		if val == float64(int64(val)) {
			buf.WriteString(strconv.FormatInt(int64(val), 10))
		} else {
			buf.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
		}
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	default:
		buf.WriteString(fmt.Sprintf("%v", v))
	}
	buf.WriteByte('\n')
}

// yamlKey returns a safely-quoted YAML mapping key.
func yamlKey(k string) string {
	if needsYAMLQuoting(k) {
		return `"` + yamlEscape(k) + `"`
	}
	return k
}

// yamlString returns a safely-quoted YAML scalar string.
func yamlString(s string) string {
	if s == "" || isYAMLReserved(s) || needsYAMLQuoting(s) {
		return `"` + yamlEscape(s) + `"`
	}
	return s
}

// needsYAMLQuoting reports whether s contains characters that have special
// meaning in YAML and therefore require the string to be quoted.
func needsYAMLQuoting(s string) bool {
	if s == "" {
		return true
	}
	if s[0] == ' ' || s[len(s)-1] == ' ' {
		return true
	}
	for _, c := range s {
		switch c {
		case ':', '#', '{', '}', '[', ']', ',', '&', '*', '?', '|',
			'-', '<', '>', '=', '!', '%', '@', '`', '"', '\'', '\\',
			'\n', '\r', '\t':
			return true
		}
	}
	return false
}

// isYAMLReserved reports whether s is a YAML reserved word.
func isYAMLReserved(s string) bool {
	switch strings.ToLower(s) {
	case "true", "false", "null", "~", "yes", "no", "on", "off":
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// yamlEscape escapes backslashes and double-quotes inside a double-quoted YAML scalar.
func yamlEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
