package encoding

import (
	"bytes"
	"encoding/json"
	"reflect"
)

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

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
