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
	return validateWithPrefix(v, "", nil)
}

// validateWithPrefix is the internal recursive validator.
// prefix is the dot-separated path to the current struct (e.g. "address").
// seen tracks visited pointer addresses to prevent infinite recursion on cyclic types.
func validateWithPrefix(v any, prefix string, seen map[uintptr]bool) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		// Cycle detection for pointer types
		ptr := rv.Pointer()
		if seen[ptr] {
			return nil
		}
		if seen == nil {
			seen = make(map[uintptr]bool)
		}
		seen[ptr] = true
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

		fieldName, _ := enc.ResolveFieldTag(sf)
		if fieldName == "-" {
			continue
		}
		qualifiedName := joinFieldPath(prefix, fieldName)

		// Dereference pointer: nil + required = fail, nil + no required = skip
		isPtr := fv.Kind() == reflect.Ptr
		if isPtr && fv.IsNil() {
			if tag != "" && containsRule(strings.Split(tag, ","), "required") {
				errs = append(errs, FieldError{Field: qualifiedName, Message: fmt.Sprintf("%s is required", qualifiedName)})
			}
			continue
		}
		if isPtr {
			// Cycle detection for pointer fields
			ptr := fv.Pointer()
			if seen[ptr] {
				continue
			}
			if seen == nil {
				seen = make(map[uintptr]bool)
			}
			seen[ptr] = true
			fv = fv.Elem()
		}

		// Recurse into nested structs
		if fv.Kind() == reflect.Struct {
			if nested := validateWithPrefix(fv.Interface(), qualifiedName, seen); nested != nil {
				var ve *ValidationError
				if errors.As(nested, &ve) {
					errs = append(errs, ve.Errors...)
				}
			}
		}

		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for ri, rule := range rules {
			rule = strings.TrimSpace(rule)

			if rule == "dive" {
				if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array {
					elemRules := rules[ri+1:]
					for j := 0; j < fv.Len(); j++ {
						elemPath := fmt.Sprintf("%s[%d]", qualifiedName, j)
						elem := fv.Index(j)
						if elem.Kind() == reflect.Ptr {
							if elem.IsNil() {
								continue
							}
							elem = elem.Elem()
						}
						if elem.Kind() == reflect.Struct {
							if nested := validateWithPrefix(elem.Interface(), elemPath, seen); nested != nil {
								var ve *ValidationError
								if errors.As(nested, &ve) {
									errs = append(errs, ve.Errors...)
								}
							}
						} else {
							for _, er := range elemRules {
								er = strings.TrimSpace(er)
								msg, ruleErr := checkRule(er, elem, elemPath)
								if ruleErr != nil {
									return ruleErr
								}
								if msg != "" {
									errs = append(errs, FieldError{Field: elemPath, Message: msg})
									break
								}
							}
						}
					}
				}
				break
			}

			msg, ruleErr := checkRule(rule, fv, qualifiedName)
			if ruleErr != nil {
				return ruleErr
			}
			if msg != "" {
				errs = append(errs, FieldError{Field: qualifiedName, Message: msg})
				break // one error per field
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// joinFieldPath joins parent prefix and field name with a dot separator.
func joinFieldPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
