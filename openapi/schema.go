package openapi

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	enc "github.com/zenqos/zenqo/internal/encoding"
)

var timeType = reflect.TypeOf(time.Time{})

type schemaBuilder struct {
	schemas  map[string]*Schema
	building map[string]bool // recursion guard for self-referential structs
}

// fromValue infers a Schema from a Go value (typically a zero-value struct).
func (sb *schemaBuilder) fromValue(v any) *Schema {
	if v == nil {
		return &Schema{Type: "object"}
	}
	return sb.fromType(reflect.TypeOf(v))
}

// fromType converts a reflect.Type to an OpenAPI Schema.
func (sb *schemaBuilder) fromType(t reflect.Type) *Schema {
	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Slices.
	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.Uint8 { // []byte → base64 string
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: sb.fromType(t.Elem())}
	}

	// time.Time → date-time string.
	if t == timeType {
		return &Schema{Type: "string", Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s := &Schema{Type: "integer"}
		if t.Kind() == reflect.Int32 {
			s.Format = "int32"
		} else if t.Kind() == reflect.Int64 {
			s.Format = "int64"
		}
		return s
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: sb.fromType(t.Elem())}
	case reflect.Struct:
		return sb.fromStruct(t)
	default:
		return &Schema{Type: "object"}
	}
}

// fromStruct converts a struct type to an OpenAPI Schema.
// Named structs are added to components/schemas and referenced via $ref.
//
// Anonymous (embedded) struct fields are inlined into the parent schema,
// mirroring Go's embedding semantics. Both value and pointer embeddings are
// handled; embedded interface fields are skipped gracefully.
func (sb *schemaBuilder) fromStruct(t reflect.Type) *Schema {
	name := t.Name()

	if name != "" {
		if _, exists := sb.schemas[name]; exists {
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		if sb.building[name] {
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		sb.building[name] = true
		defer func() { sb.building[name] = false }()
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	sb.collectStructFields(t, schema)

	if len(schema.Properties) == 0 {
		schema.Properties = nil
	}

	if name != "" {
		sb.schemas[name] = schema
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	return schema
}

// collectStructFields iterates over the fields of t, inlining any anonymous
// (embedded) struct fields directly into schema rather than nesting them.
// This matches Go's embedding semantics and produces flatter, more accurate
// OpenAPI schemas for common patterns such as gorm.Model or timestamp mixins.
func (sb *schemaBuilder) collectStructFields(t reflect.Type, schema *Schema) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		if f.Anonymous {
			// Dereference pointer embeddings: *Timestamps → Timestamps
			ft := f.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			// Skip embedded interface fields (e.g. embedded error or io.Reader).
			if ft.Kind() == reflect.Interface {
				continue
			}
			// Recurse into the embedded struct, inlining its fields.
			if ft.Kind() == reflect.Struct {
				sb.collectStructFields(ft, schema)
			}
			continue
		}

		fieldName, _ := enc.ResolveFieldTag(f)
		if fieldName == "-" {
			continue
		}

		fieldSchema := sb.fromType(f.Type)
		applyValidateTags(f, f.Type, fieldSchema, &schema.Required, fieldName)
		schema.Properties[fieldName] = fieldSchema
	}
}

// applyValidateTags reads validate:"..." struct tags and sets OpenAPI constraints
// on the field schema, and appends to the parent object's required list.
func applyValidateTags(f reflect.StructField, ft reflect.Type, schema *Schema, required *[]string, fieldName string) {
	tag := f.Tag.Get("validate")
	if tag == "" {
		return
	}
	// Dereference pointer for kind checks.
	for ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	k := ft.Kind()

	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "required":
			*required = append(*required, fieldName)
		case strings.HasPrefix(rule, "min="):
			n, err := strconv.Atoi(rule[4:])
			if err != nil {
				break
			}
			if k == reflect.String {
				schema.MinLength = &n
			} else if isNumericKind(k) {
				f64 := float64(n)
				schema.Minimum = &f64
			}
		case strings.HasPrefix(rule, "max="):
			n, err := strconv.Atoi(rule[4:])
			if err != nil {
				break
			}
			if k == reflect.String {
				schema.MaxLength = &n
			} else if isNumericKind(k) {
				f64 := float64(n)
				schema.Maximum = &f64
			}
		case rule == "email":
			schema.Format = "email"
		case rule == "url":
			schema.Format = "uri"
		case rule == "uuid":
			schema.Format = "uuid"
		case strings.HasPrefix(rule, "oneof="):
			for _, v := range strings.Split(rule[6:], "|") {
				schema.Enum = append(schema.Enum, v)
			}
		}
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
