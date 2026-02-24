package encoding

import (
	"reflect"
	"strings"
	"sync"
)

// cachedField holds pre-resolved metadata for a single struct field.
type cachedField struct {
	index     int
	key       string
	omitempty bool
	anonymous bool
	exported  bool
}

var structFieldCache sync.Map // reflect.Type → []cachedField

func getStructFields(t reflect.Type) []cachedField {
	if cached, ok := structFieldCache.Load(t); ok {
		return cached.([]cachedField)
	}
	n := t.NumField()
	fields := make([]cachedField, n)
	for i := 0; i < n; i++ {
		sf := t.Field(i)
		key, omitempty := ResolveFieldTag(sf)
		fields[i] = cachedField{
			index:     i,
			key:       key,
			omitempty: omitempty,
			anonymous: sf.Anonymous,
			exported:  sf.IsExported(),
		}
	}
	structFieldCache.Store(t, fields)
	return fields
}

// ResolveFieldTag determines the JSON key and omitempty flag for a struct field.
// Tag priority (highest to lowest):
//  1. zenqo:"key"  — Zenqo override  (zenqo:"-" to exclude a field)
//  2. json:"key"   — Standard Go tag (backward compatible)
//  3. Auto         — PascalCase field name converted to camelCase
func ResolveFieldTag(f reflect.StructField) (key string, omitempty bool) {
	// 1. zenqo tag
	if tag := f.Tag.Get("zenqo"); tag != "" {
		k, rest := splitTagName(tag)
		return k, strings.Contains(rest, "omitempty")
	}
	// 2. json tag
	if tag := f.Tag.Get("json"); tag != "" {
		k, rest := splitTagName(tag)
		if k == "" {
			k = ToCamelCase(f.Name)
		}
		return k, strings.Contains(rest, "omitempty")
	}
	// 3. auto camelCase
	return ToCamelCase(f.Name), false
}

func splitTagName(tag string) (name, rest string) {
	if i := strings.Index(tag, ","); i >= 0 {
		return tag[:i], tag[i:]
	}
	return tag, ""
}
