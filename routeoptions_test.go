package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithConstraints_Multiple(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	c1 := Int("id")
	c2 := Custom(func(rc *RequestContext, p Params) bool {
		return p["id"] != "0"
	})

	r.GET("item", "/items/{id}", h, WithConstraints(c1, c2))

	// Should match: id is int and non-zero
	req := httptest.NewRequest("GET", "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if !called {
		t.Error("expected handler to be called")
	}

	// Should not match: id is 0 (fails c2)
	called = false
	req = httptest.NewRequest("GET", "/items/0", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if called {
		t.Error("expected handler NOT to be called for id=0")
	}
}

func TestWithMetadata(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, ok := MatchFromContext(req.Context())
		if !ok {
			t.Fatal("no match in context")
		}
		if m.Route.Metadata["env"] != "prod" {
			t.Errorf("expected metadata env=prod, got %q", m.Route.Metadata["env"])
		}
	})

	r.GET("foo", "/foo", h, WithMetadata("env", "prod"))

	req := httptest.NewRequest("GET", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}

func TestWithMetadata_Multiple(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, _ := MatchFromContext(req.Context())
		if m.Route.Metadata["a"] != "1" || m.Route.Metadata["b"] != "2" {
			t.Errorf("expected metadata a=1,b=2, got %v", m.Route.Metadata)
		}
	})

	r.GET("foo", "/foo", h, WithMetadata("a", "1"), WithMetadata("b", "2"))

	req := httptest.NewRequest("GET", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}
