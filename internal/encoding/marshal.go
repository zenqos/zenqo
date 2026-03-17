package encoding

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

// Marshal encodes v to JSON with automatic PascalCase → camelCase conversion.
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
func Marshal(v any) ([]byte, error) {
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
	// encoding.TextMarshaler fallback (e.g. net/netip.Addr, net.IP)
	if v.Type().Implements(textMarshalerType) {
		text, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, err
		}
		return json.Marshal(string(text))
	}
	if v.CanAddr() && v.Addr().Type().Implements(textMarshalerType) {
		text, err := v.Addr().Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, err
		}
		return json.Marshal(string(text))
	}

	switch v.Kind() {
	case reflect.Struct:
		return encodeStruct(v)
	case reflect.Map:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return encodeMap(v)
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
	fields := getStructFields(v.Type())
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true

	for _, f := range fields {
		if !f.exported {
			continue
		}
		fv := v.Field(f.index)

		if f.anonymous {
			if err := inlineEmbedded(fv, &first, &buf); err != nil {
				return nil, err
			}
			continue
		}

		if f.key == "-" {
			continue
		}
		if f.omitempty && fv.IsZero() {
			continue
		}

		val, err := encodeValue(fv)
		if err != nil {
			return nil, err
		}
		if !first {
			buf.WriteByte(',')
		}
		writeJSONKey(&buf, f.key)
		buf.WriteByte(':')
		buf.Write(val)
		first = false
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// inlineEmbedded writes the exported fields of an embedded struct
// directly into the parent object (no nesting).
// Recursively inlines nested anonymous (embedded) fields.
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
	fields := getStructFields(v.Type())
	for _, f := range fields {
		if !f.exported {
			continue
		}
		fv := v.Field(f.index)

		// Recursively inline nested embedded structs
		if f.anonymous {
			if err := inlineEmbedded(fv, first, buf); err != nil {
				return err
			}
			continue
		}

		if f.key == "-" {
			continue
		}
		if f.omitempty && fv.IsZero() {
			continue
		}
		val, err := encodeValue(fv)
		if err != nil {
			return err
		}
		if !*first {
			buf.WriteByte(',')
		}
		writeJSONKey(buf, f.key)
		buf.WriteByte(':')
		buf.Write(val)
		*first = false
	}
	return nil
}

// encodeMap encodes a map with sorted keys, applying camelCase conversion
// to struct values within the map (unlike json.Marshal fallback).
func encodeMap(v reflect.Value) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	// Sort keys for deterministic output
	keys := v.MapKeys()
	sortedKeys := make([]reflect.Value, len(keys))
	copy(sortedKeys, keys)
	sort.Slice(sortedKeys, func(i, j int) bool {
		return fmt.Sprint(sortedKeys[i].Interface()) < fmt.Sprint(sortedKeys[j].Interface())
	})

	for i, key := range sortedKeys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// Encode key
		keyBytes, err := json.Marshal(key.Interface())
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')

		// Encode value through our pipeline (preserves camelCase for structs)
		val, err := encodeValue(v.MapIndex(key))
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
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
