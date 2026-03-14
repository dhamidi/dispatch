package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_PUT(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	if err := r.PUT("update", "/items/{id}", h); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("PUT handler not called")
	}
}

func TestRouter_PATCH(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	if err := r.PATCH("patch", "/items/{id}", h); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PATCH", "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("PATCH handler not called")
	}
}

func TestRouter_OPTIONS(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	if err := r.OPTIONS("cors", "/items", h); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("OPTIONS", "/items", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("OPTIONS handler not called")
	}
}

func TestRegisterMethod_InvalidTemplate(t *testing.T) {
	r := New()
	// An invalid URI template should return an error
	err := r.GET("bad", "/{unclosed", noopHandler)
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

func TestScope_Register_InvalidTemplate(t *testing.T) {
	r := New()
	r.Scope(func(s *Scope) {
		err := s.GET("bad", "/{unclosed", noopHandler)
		if err == nil {
			t.Error("expected error for invalid template in scope")
		}
	})
}
