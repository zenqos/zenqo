package core

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// newRequestWithParam creates a request with a chi URL param set.
func newRequestWithParam(key, value string) *http.Request {
	r, _ := http.NewRequest("GET", "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestParamString(t *testing.T) {
	r := newRequestWithParam("name", "alice")
	v, err := Param[string](r, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "alice" {
		t.Fatalf("expected alice, got %q", v)
	}
}

func TestParamInt64(t *testing.T) {
	r := newRequestWithParam("id", "42")
	v, err := Param[int64](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestParamInt(t *testing.T) {
	r := newRequestWithParam("id", "7")
	v, err := Param[int](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 7 {
		t.Fatalf("expected 7, got %d", v)
	}
}

func TestParamUint64(t *testing.T) {
	r := newRequestWithParam("id", "99")
	v, err := Param[uint64](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 99 {
		t.Fatalf("expected 99, got %d", v)
	}
}

func TestParamMissing(t *testing.T) {
	r := newRequestWithParam("other", "val")
	_, err := Param[int64](r, "id")
	if err == nil {
		t.Fatal("expected error for missing param")
	}
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != 400 {
		t.Fatalf("expected 400, got %d", he.Status)
	}
}

func TestParamInvalidInt(t *testing.T) {
	r := newRequestWithParam("id", "abc")
	_, err := Param[int64](r, "id")
	if err == nil {
		t.Fatal("expected error for non-integer param")
	}
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != 400 {
		t.Fatalf("expected 400, got %d", he.Status)
	}
}

func TestParamNegativeUint(t *testing.T) {
	r := newRequestWithParam("id", "-1")
	_, err := Param[uint64](r, "id")
	if err == nil {
		t.Fatal("expected error for negative uint param")
	}
}

// --- BindQuery edge-case tests ---

func TestBindQueryPresent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=hello", nil)
	got := BindQuery(r, "q")
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestBindQueryMissing(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search", nil)
	got := BindQuery(r, "q")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindQueryEmptyValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?key=", nil)
	got := BindQuery(r, "key")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindQueryURLEncoded(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=hello+world", nil)
	got := BindQuery(r, "q")
	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}
}

func TestBindQueryMultipleValues(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=first&q=second", nil)
	got := BindQuery(r, "q")
	if got != "first" {
		t.Fatalf("expected %q, got %q", "first", got)
	}
}

func TestBindQuerySpecialChars(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=%ED%95%9C%EA%B8%80", nil)
	got := BindQuery(r, "q")
	if got != "한글" {
		t.Fatalf("expected %q, got %q", "한글", got)
	}
}

// --- BindQueryStruct tests ---

func TestBindQueryStructBasic(t *testing.T) {
	type Q struct {
		Page  int    `query:"page"`
		Limit int    `query:"limit"`
		Sort  string `query:"sort"`
	}
	r, _ := http.NewRequest("GET", "/users?page=2&limit=10&sort=asc", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Page != 2 {
		t.Errorf("Page = %d, want 2", q.Page)
	}
	if q.Limit != 10 {
		t.Errorf("Limit = %d, want 10", q.Limit)
	}
	if q.Sort != "asc" {
		t.Errorf("Sort = %q, want %q", q.Sort, "asc")
	}
}

func TestBindQueryStructSlice(t *testing.T) {
	type Q struct {
		Tags []string `query:"tag"`
	}
	r, _ := http.NewRequest("GET", "/search?tag=go&tag=rust", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Tags) != 2 || q.Tags[0] != "go" || q.Tags[1] != "rust" {
		t.Errorf("Tags = %v, want [go rust]", q.Tags)
	}
}

func TestBindQueryStructBoolFloat(t *testing.T) {
	type Q struct {
		Active bool    `query:"active"`
		Score  float64 `query:"score"`
	}
	r, _ := http.NewRequest("GET", "/items?active=true&score=9.5", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !q.Active {
		t.Error("Active = false, want true")
	}
	if q.Score != 9.5 {
		t.Errorf("Score = %f, want 9.5", q.Score)
	}
}

func TestBindQueryStructMissing(t *testing.T) {
	type Q struct {
		Page int    `query:"page"`
		Sort string `query:"sort"`
	}
	r, _ := http.NewRequest("GET", "/users", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Page != 0 {
		t.Errorf("Page = %d, want 0", q.Page)
	}
	if q.Sort != "" {
		t.Errorf("Sort = %q, want empty", q.Sort)
	}
}

func TestBindQueryStructInvalidInt(t *testing.T) {
	type Q struct {
		Page int `query:"page"`
	}
	r, _ := http.NewRequest("GET", "/users?page=abc", nil)
	_, err := BindQueryStruct[Q](r)
	if err == nil {
		t.Fatal("expected error for invalid int")
	}
}

func TestBindQueryStructValidation(t *testing.T) {
	type Q struct {
		Limit int `query:"limit" validate:"max=100"`
	}
	r, _ := http.NewRequest("GET", "/users?limit=200", nil)
	_, err := BindQueryStruct[Q](r)
	if err == nil {
		t.Fatal("expected validation error for limit > 100")
	}
}

func TestBindQueryStructNoTag(t *testing.T) {
	type Q struct {
		Page   int `query:"page"`
		Hidden int // no query tag — should be ignored
	}
	r, _ := http.NewRequest("GET", "/users?page=1&Hidden=99", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Hidden != 0 {
		t.Errorf("Hidden = %d, want 0 (should be ignored)", q.Hidden)
	}
}

func TestBindQueryStructDashTag(t *testing.T) {
	type Q struct {
		Skip int `query:"-"`
	}
	r, _ := http.NewRequest("GET", "/users?Skip=5", nil)
	q, err := BindQueryStruct[Q](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Skip != 0 {
		t.Errorf("Skip = %d, want 0", q.Skip)
	}
}

// --- BindHeader edge-case tests ---

func TestBindHeaderPresent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer token123")
	got := BindHeader(r, "Authorization")
	if got != "Bearer token123" {
		t.Fatalf("expected %q, got %q", "Bearer token123", got)
	}
}

func TestBindHeaderMissing(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	got := BindHeader(r, "X-Missing")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindHeaderCaseInsensitive(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Content-Type", "application/json")
	got := BindHeader(r, "content-type")
	if got != "application/json" {
		t.Fatalf("expected %q, got %q", "application/json", got)
	}
}

func TestBindHeaderEmptyValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Empty", "")
	got := BindHeader(r, "X-Empty")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// ─── BindFile / BindFiles ────────────────────────────────────────────────────

// newUploadRequest builds a multipart/form-data request with one or more files.
func newUploadRequest(field string, files map[string][]byte) *http.Request {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for name, content := range files {
		fw, _ := w.CreateFormFile(field, name)
		fw.Write(content) //nolint:errcheck
	}
	w.Close()
	r, _ := http.NewRequest("POST", "/upload", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

func TestBindFile_Single(t *testing.T) {
	r := newUploadRequest("avatar", map[string][]byte{
		"photo.jpg": []byte("fake-image-data"),
	})
	f, err := BindFile(r, "avatar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Filename != "photo.jpg" {
		t.Errorf("Filename = %q, want %q", f.Filename, "photo.jpg")
	}
	if string(f.Content) != "fake-image-data" {
		t.Errorf("Content = %q, want %q", f.Content, "fake-image-data")
	}
	if f.Size != int64(len("fake-image-data")) {
		t.Errorf("Size = %d, want %d", f.Size, len("fake-image-data"))
	}
}

func TestBindFile_MissingField(t *testing.T) {
	r := newUploadRequest("document", map[string][]byte{
		"file.pdf": []byte("data"),
	})
	_, err := BindFile(r, "avatar") // wrong field
	if err == nil {
		t.Fatal("expected error for missing field, got nil")
	}
}

func TestBindFiles_Multiple(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		fw, _ := w.CreateFormFile("files", name)
		fw.Write([]byte("content-" + name)) //nolint:errcheck
	}
	w.Close()
	r, _ := http.NewRequest("POST", "/upload", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())

	files, err := BindFiles(r, "files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestBindFiles_EmptyField(t *testing.T) {
	r := newUploadRequest("images", map[string][]byte{
		"img.png": []byte("data"),
	})
	files, err := BindFiles(r, "other") // absent field
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestBindFile_DefaultContentType(t *testing.T) {
	r := newUploadRequest("file", map[string][]byte{"data.bin": []byte("bytes")})
	f, err := BindFile(r, "file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Content-Type may be application/octet-stream if not set by the writer.
	if f.ContentType == "" {
		t.Error("ContentType should not be empty")
	}
}
