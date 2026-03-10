package dispatch

import (
	"errors"
	"net/http"
	"testing"

	"github.com/dhamidi/uritemplate"
)

func mustTemplate(t *testing.T, raw string) *uritemplate.Template {
	t.Helper()
	tmpl, err := uritemplate.Parse(raw)
	if err != nil {
		t.Fatalf("uritemplate.Parse(%q): %v", raw, err)
	}
	return tmpl
}

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func TestHandle_EmptyName(t *testing.T) {
	r := New()
	err := r.Handle(Route{Name: ""})
	if !errors.Is(err, ErrEmptyRouteName) {
		t.Errorf("expected ErrEmptyRouteName, got %v", err)
	}
}

func TestHandle_DuplicateName(t *testing.T) {
	r := New()
	err := r.Handle(Route{
		Name:     "foo",
		Methods:  GET,
		Template: mustTemplate(t, "/foo"),
		Handler:  noopHandler,
	})
	if err != nil {
		t.Fatalf("first Handle: %v", err)
	}
	err = r.Handle(Route{
		Name:     "foo",
		Methods:  GET,
		Template: mustTemplate(t, "/foo"),
		Handler:  noopHandler,
	})
	if !errors.Is(err, ErrDuplicateRoute) {
		t.Errorf("expected ErrDuplicateRoute, got %v", err)
	}
}

func TestHandle_NilTemplate(t *testing.T) {
	r := New()
	err := r.Handle(Route{Name: "foo", Methods: GET, Template: nil, Handler: noopHandler})
	if !errors.Is(err, ErrNilTemplate) {
		t.Errorf("expected ErrNilTemplate, got %v", err)
	}
}

func TestHandle_NilHandler(t *testing.T) {
	r := New()
	err := r.Handle(Route{Name: "foo", Methods: GET, Template: mustTemplate(t, "/foo"), Handler: nil})
	if !errors.Is(err, ErrNilHandler) {
		t.Errorf("expected ErrNilHandler, got %v", err)
	}
}

func TestHandle_ZeroMethods(t *testing.T) {
	r := New()
	err := r.Handle(Route{Name: "foo", Methods: 0, Template: mustTemplate(t, "/foo"), Handler: noopHandler})
	if err == nil {
		t.Error("expected non-nil error for zero methods, got nil")
	}
}

func TestHandle_ValidRoute(t *testing.T) {
	r := New()
	err := r.Handle(Route{
		Name:     "foo",
		Methods:  GET,
		Template: mustTemplate(t, "/foo"),
		Handler:  noopHandler,
	})
	if err != nil {
		t.Fatalf("Handle returned unexpected error: %v", err)
	}
	route, ok := r.Route("foo")
	if !ok {
		t.Fatal("Route(\"foo\") not found after registration")
	}
	if route.Name != "foo" {
		t.Errorf("route.Name = %q, want %q", route.Name, "foo")
	}
}

func TestHandle_ClonesDefaults(t *testing.T) {
	r := New()
	defaults := Params{"x": "1"}
	err := r.Handle(Route{
		Name:     "foo",
		Methods:  GET,
		Template: mustTemplate(t, "/foo"),
		Handler:  noopHandler,
		Defaults: defaults,
	})
	if err != nil {
		t.Fatalf("Handle returned unexpected error: %v", err)
	}
	// Modify the original map after registration.
	defaults["x"] = "changed"

	route, ok := r.Route("foo")
	if !ok {
		t.Fatal("Route(\"foo\") not found after registration")
	}
	if got := route.Defaults["x"]; got != "1" {
		t.Errorf("route.Defaults[\"x\"] = %q, want %q (defaults were not cloned)", got, "1")
	}
}

func TestMustHandle_Panics(t *testing.T) {
	r := New()
	defer func() {
		if v := recover(); v == nil {
			t.Error("expected MustHandle to panic for invalid route, but it did not")
		}
	}()
	r.MustHandle(Route{Name: ""})
}

func TestMustHandle_Success(t *testing.T) {
	r := New()
	defer func() {
		if v := recover(); v != nil {
			t.Errorf("MustHandle panicked unexpectedly: %v", v)
		}
	}()
	r.MustHandle(Route{
		Name:     "foo",
		Methods:  GET,
		Template: mustTemplate(t, "/foo"),
		Handler:  noopHandler,
	})
}
