package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzCanonicalMatch(f *testing.F) {
	f.Add("/users/42", "sort=name&page=1")
	f.Add("/users/42", "")
	f.Add("/users/42", "a=1&a=2&b=3")
	f.Add("/users/42", "%00=%ff")
	f.Add("/users/1", "x=y")
	f.Add("/users/0", "")
	f.Add("/users/999", "key=value&another=thing")

	r := New(WithDefaultCanonicalPolicy(CanonicalAnnotate))
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop, WithConstraint(Int("id")))

	f.Fuzz(func(t *testing.T, path, query string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with path=%q query=%q: %v", path, query, r)
			}
		}()
		target := "http://localhost" + path
		if query != "" {
			target += "?" + query
		}
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			return
		}
		_, _ = r.Match(req)
	})
}

func FuzzCanonicalRedirect(f *testing.F) {
	f.Add("/users/42", "b=2&a=1")
	f.Add("/users/1", "")
	f.Add("/users/99", "z=1&a=2&m=3")

	r := New(WithDefaultCanonicalPolicy(CanonicalRedirect))
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop, WithConstraint(Int("id")))

	f.Fuzz(func(t *testing.T, path, query string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		target := "http://localhost" + path
		if query != "" {
			target += "?" + query
		}
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			return
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	})
}
