package dispatch

import (
	"testing"
)

func FuzzMethodSetFrom(f *testing.F) {
	f.Add("GET")
	f.Add("get")
	f.Add("PATCH")
	f.Add("PROPFIND")
	f.Add("")
	f.Add("\x00")
	f.Add("POST")
	f.Add("DELETE")
	f.Add("OPTIONS")
	f.Add("TRACE")
	f.Add("CONNECT")
	f.Add("HEAD")
	f.Add("PUT")

	f.Fuzz(func(t *testing.T, method string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic with method=%q: %v", method, r)
			}
		}()
		_, _ = MethodSetFrom(method)
		_, _ = MethodFromString(method)
	})
}
