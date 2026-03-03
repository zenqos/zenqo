package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- GuardToMiddleware tests ---

func TestGuardToMiddleware_Allowed(t *testing.T) {
	mw := GuardToMiddleware(&testGuard{allow: true})
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(next).ServeHTTP(w, r)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGuardToMiddleware_DeniedNil(t *testing.T) {
	mw := GuardToMiddleware(&testGuard{allow: false})
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(next).ServeHTTP(w, r)

	if called {
		t.Fatal("next handler should not be called")
	}
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestGuardToMiddleware_DeniedWithError(t *testing.T) {
	mw := GuardToMiddleware(&testGuard{allow: false, err: &HTTPError{Status: 500, Message: "boom"}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, r)

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGuardToMiddleware_DeniedWithHTTPError(t *testing.T) {
	mw := GuardToMiddleware(&testGuard{allow: false, err: &HTTPError{Status: 401, Message: "unauthorized"}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, r)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGuardToMiddleware_DeniedWith429(t *testing.T) {
	mw := GuardToMiddleware(&testGuard{allow: false, err: &HTTPError{Status: 429, Message: "rate limit exceeded"}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, r)

	if w.Code != 429 {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

// --- InterceptorToMiddleware tests ---

type testCtxKey string

type testInterceptor struct {
	beforeCalled bool
	afterCalled  bool
	afterStatus  int
	ctxKey       string
	ctxValue     string
}

func (i *testInterceptor) Before(ctx context.Context, r *http.Request) context.Context {
	i.beforeCalled = true
	if i.ctxKey != "" {
		return context.WithValue(ctx, testCtxKey(i.ctxKey), i.ctxValue)
	}
	return ctx
}

func (i *testInterceptor) After(ctx context.Context, w http.ResponseWriter, statusCode int) {
	i.afterCalled = true
	i.afterStatus = statusCode
}

func TestInterceptorToMiddleware_BeforeAfter(t *testing.T) {
	ic := &testInterceptor{ctxKey: "user", ctxValue: "alice"}
	mw := InterceptorToMiddleware(ic)

	var gotCtxVal interface{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtxVal = r.Context().Value("user")
		w.WriteHeader(201)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mw(next).ServeHTTP(w, r)

	if !ic.beforeCalled {
		t.Fatal("Before should be called")
	}
	if !ic.afterCalled {
		t.Fatal("After should be called")
	}
	if gotCtxVal != "alice" {
		t.Fatalf("expected context value 'alice', got %v", gotCtxVal)
	}
	if ic.afterStatus != 201 {
		t.Fatalf("expected After status 201, got %d", ic.afterStatus)
	}
}

// --- statusWriter tests ---

func TestStatusWriterCapturesCode(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, statusCode: 200}
	sw.WriteHeader(404)

	if sw.statusCode != 404 {
		t.Fatalf("expected 404, got %d", sw.statusCode)
	}
	if w.Code != 404 {
		t.Fatalf("underlying writer should also be 404, got %d", w.Code)
	}
}

func TestStatusWriterFlush(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, statusCode: 200}
	// httptest.ResponseRecorder implements http.Flusher
	sw.Flush()
	if !w.Flushed {
		t.Fatal("expected Flush to be delegated")
	}
}

type noHijackWriter struct {
	http.ResponseWriter
}

func TestStatusWriterHijackNotSupported(t *testing.T) {
	sw := &statusWriter{ResponseWriter: &noHijackWriter{httptest.NewRecorder()}, statusCode: 200}
	_, _, err := sw.Hijack()
	if err == nil {
		t.Fatal("expected error when underlying writer doesn't support hijacking")
	}
}

// Test that Hijack delegates when the underlying writer supports it.
func TestStatusWriterHijackSupported(t *testing.T) {
	done := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		sw := &statusWriter{ResponseWriter: w, statusCode: 200}
		conn, _, err := sw.Hijack()
		if err != nil {
			t.Errorf("Hijack should succeed on real server: %v", err)
			return
		}
		conn.Close()
	}))
	defer ts.Close()

	// Hijack takes over the connection, so Get will likely fail — that's expected.
	resp, err := http.Get(ts.URL)
	if err == nil {
		resp.Body.Close()
	}
	<-done
}
