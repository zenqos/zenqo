package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- adapt() tests ---

func TestAdaptGETWithData(t *testing.T) {
	h := adapt("GET", func(r *http.Request) (any, error) {
		return map[string]string{"hello": "world"}, nil
	}, DefaultErrorHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %v", body["success"])
	}
}

func TestAdaptPOSTWithData(t *testing.T) {
	h := adapt("POST", func(r *http.Request) (any, error) {
		return map[string]string{"id": "1"}, nil
	}, DefaultErrorHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	h(w, r)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestAdaptNilReturn(t *testing.T) {
	h := adapt("GET", func(r *http.Request) (any, error) {
		return nil, nil
	}, DefaultErrorHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h(w, r)

	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", w.Body.String())
	}
}

func TestAdaptErrorReturn(t *testing.T) {
	called := false
	errHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
		Error(w, 500, err.Error())
	}
	h := adapt("GET", func(r *http.Request) (any, error) {
		return nil, errors.New("something broke")
	}, errHandler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h(w, r)

	if !called {
		t.Fatal("expected errHandler to be called")
	}
	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- RegisterRoutes tests ---

type testGuard struct {
	allow bool
	err   error
}

func (g *testGuard) CanActivate(r *http.Request) (bool, error) {
	return g.allow, g.err
}

func TestRegisterRoutesControllerGuard(t *testing.T) {
	c := &BaseController{}
	c.SetBasePath("/test")
	c.UseControllerGuard(&testGuard{allow: false})
	c.GET("/hello", func(r *http.Request) (any, error) {
		return "ok", nil
	})
	c.GET("/world", func(r *http.Request) (any, error) {
		return "ok", nil
	})

	app := NewApp()
	app.UseController(c)
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// Both routes should be blocked by the controller guard
	for _, path := range []string{"/test/hello", "/test/world"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("request to %s failed: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatalf("expected 403 for %s, got %d", path, resp.StatusCode)
		}
	}
}

func TestRegisterRoutesRouteLevelGuard(t *testing.T) {
	c := &BaseController{}
	c.SetBasePath("/test")
	c.GET("/open", func(r *http.Request) (any, error) {
		return "open", nil
	})
	c.GET("/closed", func(r *http.Request) (any, error) {
		return "closed", nil
	}).UseGuard(&testGuard{allow: false})

	app := NewApp()
	app.UseController(c)
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// /open should succeed
	resp, err := http.Get(srv.URL + "/test/open")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// /closed should be blocked
	resp, err = http.Get(srv.URL + "/test/closed")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestSetBasePathPanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty basePath")
		}
	}()
	c := &BaseController{}
	c.SetBasePath("")
}

func TestSetBasePathPanicsOnMissingSlash(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for basePath without leading /")
		}
	}()
	c := &BaseController{}
	c.SetBasePath("noslash")
}

// --- #143: multiple controllers with the same base path ---

func TestMultipleControllersWithSameBasePath(t *testing.T) {
	c1 := &BaseController{}
	c1.SetBasePath("/users")
	c1.GET("/list", func(r *http.Request) (any, error) {
		return "user-list", nil
	})

	c2 := &BaseController{}
	c2.SetBasePath("/users")
	c2.GET("/portfolio", func(r *http.Request) (any, error) {
		return "portfolio-list", nil
	})

	app := NewApp()
	app.UseController(c1)
	app.UseController(c2)

	// Handler() must not panic
	var h http.Handler
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		h = app.Handler()
	}()

	srv := httptest.NewServer(h)
	defer srv.Close()

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/users/list", 200},
		{"/users/portfolio", 200},
	} {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Fatalf("GET %s: expected %d, got %d", tc.path, tc.want, resp.StatusCode)
		}
	}
}

// --- #144: SetBasePath("/") does not panic ---

func TestSetBasePathRootNoPanic(t *testing.T) {
	c := &BaseController{}
	c.SetBasePath("/")
	c.GET("/health", func(r *http.Request) (any, error) {
		return "ok", nil
	})

	app := NewApp()
	app.UseController(c)

	var h http.Handler
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		h = app.Handler()
	}()

	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSetBasePathRootWithGlobalPrefix(t *testing.T) {
	c := &BaseController{}
	c.SetBasePath("/")
	c.GET("/ping", func(r *http.Request) (any, error) {
		return "pong", nil
	})

	app := NewApp()
	app.SetGlobalPrefix("/api/v1")
	app.UseController(c)

	var h http.Handler
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		h = app.Handler()
	}()

	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/ping")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 at /api/v1/ping, got %d", resp.StatusCode)
	}
}
