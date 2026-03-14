package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzRouteMatch(f *testing.F) {
	f.Add("GET", "/users/42")
	f.Add("POST", "/posts")
	f.Add("GET", "/users/../admin")
	f.Add("GET", "/users/42?q=search&page=1")
	f.Add("DELETE", "/users/999999999999999999999")
	f.Add("GET", "/%2e%2e/admin")
	f.Add("GET", "/users/42/")
	f.Add("GET", "/")
	f.Add("GET", "/search?q=hello")
	f.Add("", "/users")

	r := New()
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.index", "/users", noop)
	r.GET("users.show", "/users/{id}", noop, WithConstraint(Int("id")))
	r.POST("users.create", "/users", noop)
	r.GET("posts.index", "/posts", noop)
	r.GET("search", "/search{?q}", noop)

	f.Fuzz(func(t *testing.T, method, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with method=%q path=%q: %v", method, path, r)
			}
		}()
		req, err := http.NewRequest(method, "http://localhost"+path, nil)
		if err != nil {
			return // invalid input for net/http, skip
		}
		_, _ = r.Match(req)
	})
}

func FuzzRouteMatchRawPath(f *testing.F) {
	f.Add("GET", "/users/42")
	f.Add("GET", "/users/hello%20world")
	f.Add("GET", "/users/%00")
	f.Add("GET", "/a/b/c/d/e/f/g")

	r := New()
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop)
	r.GET("deep", "/a/b/c/d/e/f/g", noop)

	f.Fuzz(func(t *testing.T, method, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		req, err := http.NewRequest(method, "http://localhost"+path, nil)
		if err != nil {
			return
		}
		_, _ = r.Match(req)
	})
}

func FuzzRouteMatchRecorder(f *testing.F) {
	f.Add("GET", "/users/42")
	f.Add("POST", "/users")
	f.Add("DELETE", "/users/42")

	r := New()
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop, WithConstraint(Int("id")))
	r.POST("users.create", "/users", noop)
	r.DELETE("users.delete", "/users/{id}", noop, WithConstraint(Int("id")))

	f.Fuzz(func(t *testing.T, method, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		req := httptest.NewRequest(method, "http://localhost"+path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	})
}
