package dispatch

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestMatch_BasicGet(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler); err != nil {
		t.Fatalf("registration: %v", err)
	}

	m, err := r.Match(httptest.NewRequest("GET", "/users/42", nil))
	if err != nil {
		t.Fatalf("Match returned unexpected error: %v", err)
	}
	if m.Name != "users.show" {
		t.Errorf("Name = %q, want %q", m.Name, "users.show")
	}
	if m.Params["id"] != "42" {
		t.Errorf("Params[\"id\"] = %q, want %q", m.Params["id"], "42")
	}
	if m.Method != "GET" {
		t.Errorf("Method = %q, want %q", m.Method, "GET")
	}
}

func TestMatch_MethodFilter(t *testing.T) {
	r := New()
	if err := r.GET("foo", "/foo", noopHandler); err != nil {
		t.Fatalf("registration: %v", err)
	}

	_, err := r.Match(httptest.NewRequest("POST", "/foo", nil))
	if !errors.Is(err, ErrMethodNotAllowed) {
		t.Errorf("expected ErrMethodNotAllowed, got %v", err)
	}
}

func TestMatch_HeadSatisfiedByGet(t *testing.T) {
	r := New()
	if err := r.GET("bar", "/bar", noopHandler); err != nil {
		t.Fatalf("registration: %v", err)
	}

	m, err := r.Match(httptest.NewRequest("HEAD", "/bar", nil))
	if err != nil {
		t.Fatalf("Match returned unexpected error: %v", err)
	}
	if m.Name != "bar" {
		t.Errorf("Name = %q, want %q", m.Name, "bar")
	}
}

func TestMatch_ConstraintRejection(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler, WithConstraint(Int("id"))); err != nil {
		t.Fatalf("registration: %v", err)
	}

	_, err := r.Match(httptest.NewRequest("GET", "/users/abc", nil))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMatch_MultiCandidateScoring(t *testing.T) {
	r := New()
	if err := r.GET("items.new", "/items/new", noopHandler); err != nil {
		t.Fatalf("registration items.new: %v", err)
	}
	if err := r.GET("items.show", "/items/{id}", noopHandler); err != nil {
		t.Fatalf("registration items.show: %v", err)
	}

	// Literal "new" should beat variable {id}
	m, err := r.Match(httptest.NewRequest("GET", "/items/new", nil))
	if err != nil {
		t.Fatalf("Match(/items/new) error: %v", err)
	}
	if m.Name != "items.new" {
		t.Errorf("Name = %q, want %q", m.Name, "items.new")
	}

	// Non-literal should match variable route
	m, err = r.Match(httptest.NewRequest("GET", "/items/42", nil))
	if err != nil {
		t.Fatalf("Match(/items/42) error: %v", err)
	}
	if m.Name != "items.show" {
		t.Errorf("Name = %q, want %q", m.Name, "items.show")
	}
}

func TestMatch_MultiCandidateDeterminism(t *testing.T) {
	type regEntry struct {
		name string
		tmpl string
	}

	orders := [][]regEntry{
		{{"items.new", "/items/new"}, {"items.show", "/items/{id}"}},
		{{"items.show", "/items/{id}"}, {"items.new", "/items/new"}},
	}

	for i, order := range orders {
		r := New()
		for _, e := range order {
			if err := r.GET(e.name, e.tmpl, noopHandler); err != nil {
				t.Fatalf("order %d: registration %q: %v", i, e.name, err)
			}
		}

		m, err := r.Match(httptest.NewRequest("GET", "/items/new", nil))
		if err != nil {
			t.Fatalf("order %d: Match error: %v", i, err)
		}
		if m.Name != "items.new" {
			t.Errorf("order %d: Name = %q, want %q (literal should beat variable regardless of order)", i, m.Name, "items.new")
		}
	}
}

func TestMatch_QueryStrictRejectsExtra(t *testing.T) {
	r := New()
	if err := r.GET("search", "/search{?q}", noopHandler, WithQueryMode(QueryStrict)); err != nil {
		t.Fatalf("registration: %v", err)
	}

	_, err := r.Match(httptest.NewRequest("GET", "/search?q=go&extra=1", nil))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMatch_QueryStrictAcceptsDeclared(t *testing.T) {
	r := New()
	if err := r.GET("search", "/search{?q}", noopHandler, WithQueryMode(QueryStrict)); err != nil {
		t.Fatalf("registration: %v", err)
	}

	m, err := r.Match(httptest.NewRequest("GET", "/search?q=go", nil))
	if err != nil {
		t.Fatalf("Match returned unexpected error: %v", err)
	}
	if m.Params["q"] != "go" {
		t.Errorf("Params[\"q\"] = %q, want %q", m.Params["q"], "go")
	}
}

func TestMatch_QueryLooseAcceptsExtra(t *testing.T) {
	r := New()
	if err := r.GET("search", "/search{?q}", noopHandler, WithQueryMode(QueryLoose)); err != nil {
		t.Fatalf("registration: %v", err)
	}

	_, err := r.Match(httptest.NewRequest("GET", "/search?q=go&extra=1", nil))
	if err != nil {
		t.Fatalf("Match returned unexpected error: %v", err)
	}
}

func TestMatch_NotFound(t *testing.T) {
	r := New()
	if err := r.GET("known", "/known", noopHandler); err != nil {
		t.Fatalf("registration: %v", err)
	}

	_, err := r.Match(httptest.NewRequest("GET", "/unknown", nil))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMatch_DefaultsApplied(t *testing.T) {
	r := New()
	if err := r.GET("page", "/page{/name}", noopHandler, WithDefaults(Params{"name": "home"})); err != nil {
		t.Fatalf("registration: %v", err)
	}

	m, err := r.Match(httptest.NewRequest("GET", "/page", nil))
	if err != nil {
		t.Fatalf("Match returned unexpected error: %v", err)
	}
	if m.Params["name"] != "home" {
		t.Errorf("Params[\"name\"] = %q, want %q", m.Params["name"], "home")
	}
}
