package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	enc "github.com/zenqos/zenqo/internal/encoding"
)

// validate checks struct fields annotated with `validate:"..."` tags.
// Returns a *ValidationError if any field fails, nil otherwise.
// Non-struct types are silently skipped. Nested structs are validated recursively.
func validate(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	rt := rv.Type()
	var errs []FieldError

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := rv.Field(i)

		tag := sf.Tag.Get("validate")

		// Dereference pointer: nil + required = fail, nil + no required = skip
		isPtr := fv.Kind() == reflect.Ptr
		if isPtr && fv.IsNil() {
			if tag != "" {
				fieldName, _ := enc.ResolveFieldTag(sf)
				if fieldName != "-" && containsRule(strings.Split(tag, ","), "required") {
					errs = append(errs, FieldError{Field: fieldName, Message: fmt.Sprintf("%s is required", fieldName)})
				}
			}
			continue
		}
		if isPtr {
			fv = fv.Elem()
		}

		// Recurse into nested structs
		if fv.Kind() == reflect.Struct {
			if nested := validate(fv.Interface()); nested != nil {
				var ve *ValidationError
				if errors.As(nested, &ve) {
					errs = append(errs, ve.Errors...)
				}
			}
		}

		if tag == "" {
			continue
		}

		fieldName, _ := enc.ResolveFieldTag(sf)
		if fieldName == "-" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if msg := checkRule(rule, fv, fieldName); msg != "" {
				errs = append(errs, FieldError{Field: fieldName, Message: msg})
				break // one error per field
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}
