package dispatch

import (
	"net/http"
	"testing"
)

func FuzzRouteRegistration(f *testing.F) {
	f.Add("/users/{id}")
	f.Add("/search{?q,page}")
	f.Add("/{+path}")
	f.Add("/{#fragment}")
	f.Add("/a{.format}")
	f.Add("/{;params}")
	f.Add("/{{nested}}")
	f.Add("")
	f.Add("/{")
	f.Add("/}")
	f.Add("/users/{id}/posts/{post_id}")
	f.Add("/{a}/{b}/{c}/{d}")
	f.Add("/static/path")
	f.Add("/{id}{?q}")
	f.Add("/\x00/\xff")

	f.Fuzz(func(t *testing.T, tmpl string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with tmpl=%q: %v", tmpl, r)
			}
		}()
		r := New()
		noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		// Must not panic (errors are acceptable)
		_ = r.GET("test", tmpl, noop)
	})
}
