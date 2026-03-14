package dispatch

import (
	"net/http"
	"testing"
)

func FuzzURLGeneration(f *testing.F) {
	f.Add("hello")
	f.Add("foo/bar")
	f.Add("a b c")
	f.Add("../../etc/passwd")
	f.Add("<script>alert(1)</script>")
	f.Add("%00%ff")
	f.Add("")
	f.Add("with spaces")
	f.Add("special!@#$%^&*()")
	f.Add("\x00")
	f.Add("very-long-" + string(make([]byte, 1000)))

	r := New()
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop)
	r.GET("search", "/search{?q}", noop)

	f.Fuzz(func(t *testing.T, val string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with val=%q: %v", val, r)
			}
		}()
		_, _ = r.URL("users.show", Params{"id": val})
		_, _ = r.Path("users.show", Params{"id": val})
		_, _ = r.URL("search", Params{"q": val})
		_, _ = r.Path("search", Params{"q": val})
	})
}

func FuzzURLGenerationUnknownRoute(f *testing.F) {
	f.Add("nonexistent")
	f.Add("")
	f.Add("users.show")

	r := New()
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", noop)

	f.Fuzz(func(t *testing.T, name string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		_, _ = r.URL(name, Params{"id": "1"})
		_, _ = r.Path(name, Params{"id": "1"})
	})
}
