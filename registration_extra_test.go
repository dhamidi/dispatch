package dispatch

import (
	"net/http"
	"testing"

	"github.com/dhamidi/uritemplate"
)

func TestHandle_RedirectCodeValidation(t *testing.T) {
	r := New()
	tmpl := uritemplate.MustParse("/test")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Valid redirect code
	err := r.Handle(Route{
		Name:            "good",
		Methods:         GET,
		Template:        tmpl,
		Handler:         h,
		CanonicalPolicy: CanonicalRedirect,
		RedirectCode:    301,
	})
	if err != nil {
		t.Errorf("expected no error for valid redirect code, got %v", err)
	}

	// Invalid redirect code (non-3xx)
	err = r.Handle(Route{
		Name:            "bad",
		Methods:         GET,
		Template:        uritemplate.MustParse("/bad"),
		Handler:         h,
		CanonicalPolicy: CanonicalRedirect,
		RedirectCode:    200,
	})
	if err == nil {
		t.Error("expected error for non-3xx redirect code")
	}

	// RedirectCode 0 with CanonicalRedirect is fine (uses default)
	err = r.Handle(Route{
		Name:            "zero",
		Methods:         GET,
		Template:        uritemplate.MustParse("/zero"),
		Handler:         h,
		CanonicalPolicy: CanonicalRedirect,
		RedirectCode:    0,
	})
	if err != nil {
		t.Errorf("expected no error for zero redirect code, got %v", err)
	}
}

func TestHandle_ClonesMetadata(t *testing.T) {
	r := New()
	meta := map[string]string{"key": "original"}
	tmpl := uritemplate.MustParse("/test")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	err := r.Handle(Route{
		Name:     "test",
		Methods:  GET,
		Template: tmpl,
		Handler:  h,
		Metadata: meta,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Mutate original map - should not affect registered route
	meta["key"] = "mutated"

	route, ok := r.Route("test")
	if !ok {
		t.Fatal("route not found")
	}
	if route.Metadata["key"] != "original" {
		t.Errorf("expected original, got %q", route.Metadata["key"])
	}
}

func TestHandle_ClonesDefaults_Isolation(t *testing.T) {
	r := New()
	defaults := Params{"key": "original"}
	tmpl := uritemplate.MustParse("/test")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	err := r.Handle(Route{
		Name:     "clone_defaults",
		Methods:  GET,
		Template: tmpl,
		Handler:  h,
		Defaults: defaults,
	})
	if err != nil {
		t.Fatal(err)
	}

	defaults["key"] = "mutated"

	route, ok := r.Route("clone_defaults")
	if !ok {
		t.Fatal("route not found")
	}
	if route.Defaults["key"] != "original" {
		t.Errorf("expected original, got %q", route.Defaults["key"])
	}
}

func TestHandle_ClonesConstraints(t *testing.T) {
	r := New()
	constraints := []Constraint{Int("id")}
	tmpl := uritemplate.MustParse("/test/{id}")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	err := r.Handle(Route{
		Name:        "clone_constraints",
		Methods:     GET,
		Template:    tmpl,
		Handler:     h,
		Constraints: constraints,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Mutate original slice
	constraints[0] = nil

	route, ok := r.Route("clone_constraints")
	if !ok {
		t.Fatal("route not found")
	}
	if route.Constraints[0] == nil {
		t.Error("constraint should not be nil after mutation of original")
	}
}
