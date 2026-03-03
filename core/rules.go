package core

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

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
	case rule == "url":
		return checkURL(fv, field)
	case rule == "uuid":
		return checkUUID(fv, field)
	case rule == "alpha":
		return checkAlpha(fv, field)
	case rule == "alphanum":
		return checkAlphaNum(fv, field)
	case rule == "numeric":
		return checkNumeric(fv, field)
	case strings.HasPrefix(rule, "len="):
		return checkLen(rule[4:], fv, field)
	case strings.HasPrefix(rule, "regex="):
		return checkRegex(rule[6:], fv, field)
	case strings.HasPrefix(rule, "contains="):
		return checkContains(rule[9:], fv, field)
	case strings.HasPrefix(rule, "startswith="):
		return checkStartsWith(rule[11:], fv, field)
	case strings.HasPrefix(rule, "endswith="):
		return checkEndsWith(rule[9:], fv, field)
	case rule == "lowercase":
		return checkLowercase(fv, field)
	case rule == "uppercase":
		return checkUppercase(fv, field)
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

func checkURL(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Sprintf("%s must be a valid URL", field)
	}
	return ""
}

func checkUUID(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if !uuidRegex.MatchString(s) {
		return fmt.Sprintf("%s must be a valid UUID", field)
	}
	return ""
}

func checkAlpha(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return fmt.Sprintf("%s must contain only letters", field)
		}
	}
	return ""
}

func checkAlphaNum(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return fmt.Sprintf("%s must contain only letters and numbers", field)
		}
	}
	return ""
}

func checkNumeric(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return fmt.Sprintf("%s must contain only numbers", field)
		}
	}
	return ""
}

func checkLen(param string, fv reflect.Value, field string) string {
	n, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}
	switch fv.Kind() {
	case reflect.String:
		if fv.Len() != n {
			return fmt.Sprintf("%s must be exactly %d characters", field, n)
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if fv.Len() != n {
			return fmt.Sprintf("%s must have exactly %d items", field, n)
		}
	}
	return ""
}

func checkRegex(pattern string, fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Sprintf("%s has invalid regex pattern", field)
	}
	if !re.MatchString(s) {
		return fmt.Sprintf("%s must match pattern %s", field, pattern)
	}
	return ""
}

func checkContains(substr string, fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if !strings.Contains(s, substr) {
		return fmt.Sprintf("%s must contain %q", field, substr)
	}
	return ""
}

func checkStartsWith(prefix string, fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, prefix) {
		return fmt.Sprintf("%s must start with %q", field, prefix)
	}
	return ""
}

func checkEndsWith(suffix string, fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, suffix) {
		return fmt.Sprintf("%s must end with %q", field, suffix)
	}
	return ""
}

func checkLowercase(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if s != strings.ToLower(s) {
		return fmt.Sprintf("%s must be lowercase", field)
	}
	return ""
}

func checkUppercase(fv reflect.Value, field string) string {
	if fv.Kind() != reflect.String {
		return ""
	}
	s := fv.String()
	if s == "" {
		return ""
	}
	if s != strings.ToUpper(s) {
		return fmt.Sprintf("%s must be uppercase", field)
	}
	return ""
}

func containsRule(rules []string, target string) bool {
	for _, r := range rules {
		if strings.TrimSpace(r) == target {
			return true
		}
	}
	return false
}
