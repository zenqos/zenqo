package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func checkRule(rule string, fv reflect.Value, field string) string {
	switch {
	case rule == "required":
		return checkRequired(fv, field)
	case strings.HasPrefix(rule, "min="):
		return checkMin(rule[4:], fv, field)
	case strings.HasPrefix(rule, "max="):
		return checkMax(rule[4:], fv, field)
	case rule == "email":
		return checkEmail(fv, field)
	case strings.HasPrefix(rule, "oneof="):
		return checkOneOf(rule[6:], fv, field)
	}
	return ""
}

func checkRequired(fv reflect.Value, field string) string {
	if fv.IsZero() {
		return fmt.Sprintf("%s is required", field)
	}
	return ""
}

func checkMin(param string, fv reflect.Value, field string) string {
	n, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}
	switch fv.Kind() {
	case reflect.String:
		if fv.Len() < n {
			return fmt.Sprintf("%s must be at least %d characters", field, n)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fv.Int() < int64(n) {
			return fmt.Sprintf("%s must be at least %d", field, n)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if fv.Uint() < uint64(n) {
			return fmt.Sprintf("%s must be at least %d", field, n)
		}
	case reflect.Float32, reflect.Float64:
		if fv.Float() < float64(n) {
			return fmt.Sprintf("%s must be at least %d", field, n)
		}
	}
	return ""
}

func checkMax(param string, fv reflect.Value, field string) string {
	n, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}
	switch fv.Kind() {
	case reflect.String:
		if fv.Len() > n {
			return fmt.Sprintf("%s must be at most %d characters", field, n)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fv.Int() > int64(n) {
			return fmt.Sprintf("%s must be at most %d", field, n)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if fv.Uint() > uint64(n) {
			return fmt.Sprintf("%s must be at most %d", field, n)
		}
	case reflect.Float32, reflect.Float64:
		if fv.Float() > float64(n) {
			return fmt.Sprintf("%s must be at most %d", field, n)
		}
	}
	return ""
}

func checkEmail(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return "" // empty + email = pass (use required separately)
	}
	if !emailRegex.MatchString(s) {
		return fmt.Sprintf("%s must be a valid email address", field)
	}
	return ""
}

func checkOneOf(param string, fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	allowed := strings.Split(param, "|")
	for _, a := range allowed {
		if s == a {
			return ""
		}
	}
	return fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", "))
}

func containsRule(rules []string, target string) bool {
	for _, r := range rules {
		if strings.TrimSpace(r) == target {
			return true
		}
	}
	return false
}
