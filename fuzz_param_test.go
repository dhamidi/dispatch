package dispatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildRequestWithParam creates a request whose context contains a Match with
// the given parameter name and value, simulating what the router does after
// successful matching.
func buildRequestWithParam(name, value string) *http.Request {
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	m := &Match{
		Params: Params{name: value},
	}
	ctx := context.WithValue(req.Context(), contextKeyMatch{}, m)
	return req.WithContext(ctx)
}

func FuzzParamExtraction(f *testing.F) {
	f.Add("42")
	f.Add("-1")
	f.Add("99999999999999999999999")
	f.Add("3.14")
	f.Add("true")
	f.Add("NaN")
	f.Add("")
	f.Add("\x00\xff")
	f.Add("yes")
	f.Add("no")
	f.Add("0")
	f.Add("1")
	f.Add("false")
	f.Add("9223372036854775807")
	f.Add("-9223372036854775808")
	f.Add("1.7976931348623157e+308")

	f.Fuzz(func(t *testing.T, raw string) {
		r := buildRequestWithParam("id", raw)
		// None of these should panic
		_, _ = ParamInt(r, "id")
		_, _ = ParamInt64(r, "id")
		_, _ = ParamFloat64(r, "id")
		_, _ = ParamBool(r, "id")
		_, _ = ParamString(r, "id")
	})
}

func FuzzParamExtractionMissing(f *testing.F) {
	f.Add("id")
	f.Add("name")
	f.Add("")
	f.Add("nonexistent")

	f.Fuzz(func(t *testing.T, name string) {
		// Request with no match context at all
		req := httptest.NewRequest("GET", "http://localhost/test", nil)
		_, _ = ParamInt(req, name)
		_, _ = ParamInt64(req, name)
		_, _ = ParamFloat64(req, name)
		_, _ = ParamBool(req, name)
		_, _ = ParamString(req, name)
	})
}
