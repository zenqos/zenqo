package core

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"unicode"
)

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// zMarshal encodes v to JSON with automatic PascalCase → camelCase conversion.
// Struct tags are completely optional.
//
// Tag priority (highest to lowest):
//  1. zenqo:"key"  — Zenqo override  (zenqo:"-" to exclude a field)
//  2. json:"key"   — Standard Go tag (backward compatible)
//  3. Auto         — PascalCase field name converted to camelCase
//
// Examples (no tags needed):
//
//	ID        → "id"
//	Name      → "name"
//	CreatedAt → "createdAt"
//	UserID    → "userId"
func zMarshal(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return encodeValue(reflect.ValueOf(v))
}

func encodeValue(v reflect.Value) ([]byte, error) {
	// Unwrap pointer and check json.Marshaler (e.g. time.Time)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return []byte("null"), nil
		}
		if v.Type().Implements(jsonMarshalerType) {
			return v.Interface().(json.Marshaler).MarshalJSON()
		}
		v = v.Elem()
	}
	if v.Type().Implements(jsonMarshalerType) {
		return v.Interface().(json.Marshaler).MarshalJSON()
	}
	if v.CanAddr() && v.Addr().Type().Implements(jsonMarshalerType) {
		return v.Addr().Interface().(json.Marshaler).MarshalJSON()
	}

	switch v.Kind() {
	case reflect.Struct:
		return encodeStruct(v)
	case reflect.Slice:
		if v.IsNil() {
			return []byte("[]"), nil
		}
		return encodeSlice(v)
	case reflect.Array:
		return encodeSlice(v)
	case reflect.Interface:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return encodeValue(v.Elem())
	default:
		return json.Marshal(v.Interface())
	}
}

func encodeStruct(v reflect.Value) ([]byte, error) {
	t := v.Type()
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		if !sf.IsExported() {
			continue
		}

		// Embedded struct: inline its exported fields
		if sf.Anonymous {
			if err := inlineEmbedded(fv, &first, &buf); err != nil {
				return nil, err
			}
			continue
		}

		key, omitempty := resolveFieldTag(sf)
		if key == "-" {
			continue
		}
		if omitempty && fv.IsZero() {
			continue
		}

		val, err := encodeValue(fv)
		if err != nil {
			return nil, err
		}
		if !first {
			buf.WriteByte(',')
		}
		writeJSONKey(&buf, key)
		buf.WriteByte(':')
		buf.Write(val)
		first = false
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// inlineEmbedded writes the exported fields of an embedded struct
// directly into the parent object (no nesting).
func inlineEmbedded(v reflect.Value, first *bool, buf *bytes.Buffer) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)
		if !sf.IsExported() {
			continue
		}
		key, omitempty := resolveFieldTag(sf)
		if key == "-" {
			continue
		}
		if omitempty && fv.IsZero() {
			continue
		}
		val, err := encodeValue(fv)
		if err != nil {
			return err
		}
		if !*first {
			buf.WriteByte(',')
		}
		writeJSONKey(buf, key)
		buf.WriteByte(':')
		buf.Write(val)
		*first = false
	}
	return nil
}

func encodeSlice(v reflect.Value) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, err := encodeValue(v.Index(i))
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func writeJSONKey(buf *bytes.Buffer, key string) {
	b, _ := json.Marshal(key)
	buf.Write(b)
}

func resolveFieldTag(f reflect.StructField) (key string, omitempty bool) {
	// 1. zenqo tag
	if tag := f.Tag.Get("zenqo"); tag != "" {
		k, rest := splitTagName(tag)
		return k, strings.Contains(rest, "omitempty")
	}
	// 2. json tag
	if tag := f.Tag.Get("json"); tag != "" {
		k, rest := splitTagName(tag)
		if k == "" {
			k = toCamelCase(f.Name)
		}
		return k, strings.Contains(rest, "omitempty")
	}
	// 3. auto camelCase
	return toCamelCase(f.Name), false
}

func splitTagName(tag string) (name, rest string) {
	if i := strings.Index(tag, ","); i >= 0 {
		return tag[:i], tag[i:]
	}
	return tag, ""
}

// toCamelCase converts a Go PascalCase identifier to camelCase.
//
//	ID          → id
//	Name        → name
//	CreatedAt   → createdAt
//	UserID      → userId
//	HTMLContent → htmlContent
func toCamelCase(s string) string {
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
