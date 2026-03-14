package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzServeHTTP(f *testing.F) {
	f.Add("GET", "/users/42")
	f.Add("GET", "/users/42/")
	f.Add("POST", "/users")
	f.Add("GET", "/nonexistent")
	f.Add("DELETE", "/users/1")
	f.Add("HEAD", "/users/42")
	f.Add("OPTIONS", "/users")
	f.Add("GET", "/")
	f.Add("GET", "/users")

	r := New(WithDefaultSlashPolicy(SlashRedirect))
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.GET("users.index", "/users", noop)
	r.GET("users.show", "/users/{id}", noop, WithConstraint(Int("id")))
	r.POST("users.create", "/users", noop)
	r.DELETE("users.delete", "/users/{id}", noop, WithConstraint(Int("id")))

	f.Fuzz(func(t *testing.T, method, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with method=%q path=%q: %v", method, path, r)
			}
		}()
		req, err := http.NewRequest(method, "http://localhost"+path, nil)
		if err != nil {
			return
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	})
}

func FuzzServeHTTPComplex(f *testing.F) {
	f.Add("GET", "/posts/hello")
	f.Add("GET", "/search?q=test")
	f.Add("PUT", "/posts/hello")
	f.Add("GET", "/posts/hello/")

	r := New(
		WithDefaultSlashPolicy(SlashRedirect),
		WithDefaultCanonicalPolicy(CanonicalAnnotate),
	)
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.GET("posts.show", "/posts/{slug}", noop)
	r.PUT("posts.update", "/posts/{slug}", noop)
	r.GET("search", "/search{?q}", noop)

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
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	})
}
