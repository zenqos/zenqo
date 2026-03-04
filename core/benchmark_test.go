package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	enc "github.com/zenqos/zenqo/internal/encoding"
)

// --- chi raw handler vs Zenqo adapt() overhead ---

func BenchmarkChiRawHandler(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"success":true,"data":{"message":"hello"}}`)) //nolint:errcheck
	})
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkZenqoAdapt(b *testing.B) {
	handler := adapt("GET", func(r *http.Request) (any, error) {
		return map[string]string{"message": "hello"}, nil
	}, DefaultErrorHandler)
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkZenqoAdaptPOST(b *testing.B) {
	handler := adapt("POST", func(r *http.Request) (any, error) {
		return map[string]string{"id": "1"}, nil
	}, DefaultErrorHandler)
	r := httptest.NewRequest("POST", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkZenqoAdaptNil(b *testing.B) {
	handler := adapt("DELETE", func(r *http.Request) (any, error) {
		return nil, nil
	}, DefaultErrorHandler)
	r := httptest.NewRequest("DELETE", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

// --- JSON serialization: encoding/json vs enc.Marshal ---

type benchStruct struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt string
	IsActive  bool
}

var benchData = benchStruct{
	ID:        42,
	Name:      "Alice Kim",
	Email:     "alice@example.com",
	CreatedAt: "2025-01-01T00:00:00Z",
	IsActive:  true,
}

func BenchmarkStdJSONMarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(benchData) //nolint:errcheck
	}
}

func BenchmarkZenqoMarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		enc.Marshal(benchData) //nolint:errcheck
	}
}

func BenchmarkStdJSONMarshalNested(b *testing.B) {
	type nested struct {
		User  benchStruct
		Items []string
		Meta  map[string]int
	}
	data := nested{
		User:  benchData,
		Items: []string{"a", "b", "c"},
		Meta:  map[string]int{"total": 100, "page": 1},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		json.Marshal(data) //nolint:errcheck
	}
}

func BenchmarkZenqoMarshalNested(b *testing.B) {
	type nested struct {
		User  benchStruct
		Items []string
		Meta  map[string]int
	}
	data := nested{
		User:  benchData,
		Items: []string{"a", "b", "c"},
		Meta:  map[string]int{"total": 100, "page": 1},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		enc.Marshal(data) //nolint:errcheck
	}
}

// --- Validation ---

type benchDTO struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Role  string `validate:"oneof=admin|user|guest"`
	Age   int    `validate:"min=18,max=120"`
}

func BenchmarkValidatePass(b *testing.B) {
	dto := benchDTO{Name: "Alice", Email: "alice@example.com", Role: "admin", Age: 25}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		validate(dto) //nolint:errcheck
	}
}

func BenchmarkValidateFail(b *testing.B) {
	dto := benchDTO{Name: "", Email: "bad", Role: "invalid", Age: 10}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		validate(dto) //nolint:errcheck
	}
}

// --- Guard middleware ---

func BenchmarkGuardAllow(b *testing.B) {
	mw := GuardToMiddleware(&testGuard{allow: true})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := mw(next)
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkGuardDeny(b *testing.B) {
	mw := GuardToMiddleware(&testGuard{allow: false})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := mw(next)
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkGuardChain3(b *testing.B) {
	g := &testGuard{allow: true}
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	for i := 0; i < 3; i++ {
		handler = GuardToMiddleware(g)(handler)
	}
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
