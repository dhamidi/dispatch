package dispatch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhamidi/uritemplate"
)

func TestParamsHelpers(t *testing.T) {
	p := Params{"id": "42", "name": "alice"}
	if p.Get("id") != "42" {
		t.Error("Get failed")
	}
	if p.Get("missing") != "" {
		t.Error("Get missing should be empty")
	}
	v, ok := p.Lookup("name")
	if !ok || v != "alice" {
		t.Error("Lookup failed")
	}
	_, ok = p.Lookup("missing")
	if ok {
		t.Error("Lookup missing should be false")
	}
	c := p.Clone()
	c["id"] = "99"
	if p["id"] != "42" {
		t.Error("Clone should not alias")
	}
}

func TestMethodFromString(t *testing.T) {
	m, err := MethodFromString("GET")
	if err != nil || m != GET {
		t.Error("MethodFromString GET failed")
	}
	_, err = MethodFromString("get")
	if err == nil {
		t.Error("MethodFromString should reject lowercase")
	}
	_, err = MethodFromString("UNKNOWN")
	if err == nil {
		t.Error("MethodFromString should reject unknown")
	}
}

func TestMethodSetContains(t *testing.T) {
	ms := GET | POST
	if !ms.Has(GET) {
		t.Error("should contain GET")
	}
	if ms.Has(DELETE) {
		t.Error("should not contain DELETE")
	}
}

func TestCandidateScoreBeats(t *testing.T) {
	a := candidateScore{LiteralSegments: 5, Registration: 0}
	b := candidateScore{LiteralSegments: 3, Registration: 1}
	if !a.beats(b) {
		t.Error("a should beat b (more literal segments)")
	}
}

func TestRouterHandleValidation(t *testing.T) {
	r := New()
	tmpl := uritemplate.MustParse("/test")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	if err := r.Handle(Route{}); err != ErrEmptyRouteName {
		t.Error("expected ErrEmptyRouteName")
	}
	if err := r.Handle(Route{Name: "test"}); err != ErrNilTemplate {
		t.Error("expected ErrNilTemplate")
	}
	if err := r.Handle(Route{Name: "test", Template: tmpl}); err != ErrNilHandler {
		t.Error("expected ErrNilHandler")
	}
	if err := r.Handle(Route{Name: "test", Template: tmpl, Handler: h}); err == nil {
		t.Error("expected error for zero method set")
	}
	if err := r.Handle(Route{Name: "test", Template: tmpl, Handler: h, Methods: GET}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := r.Handle(Route{Name: "test", Template: tmpl, Handler: h, Methods: GET}); !errors.Is(err, ErrDuplicateRoute) {
		t.Error("expected ErrDuplicateRoute")
	}
}

func TestRouterGETAndMatch(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	if err := r.GET("home", "/", h); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "home" {
		t.Errorf("expected home, got %s", m.Name)
	}
	_ = called
}

func TestRouterMatchWithParams(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	if err := r.GET("users.show", "/users/{id}", h); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/users/42", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Params.Get("id") != "42" {
		t.Errorf("expected id=42, got %s", m.Params.Get("id"))
	}
}

func TestRouterMatchDefaults(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	if err := r.GET("users.show", "/users/{id}", h,
		WithDefaults(Params{"format": "html"})); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/users/42", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Params.Get("format") != "html" {
		t.Errorf("expected format=html, got %s", m.Params.Get("format"))
	}
	if m.Params.Get("id") != "42" {
		t.Errorf("default should not override extracted: got id=%s", m.Params.Get("id"))
	}
}

func TestRouterNotFound(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("home", "/", h)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	_, err := r.Match(req)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("home", "/", h)
	req := httptest.NewRequest("POST", "/", nil)
	_, err := r.Match(req)
	if err != ErrMethodNotAllowed {
		t.Errorf("expected ErrMethodNotAllowed, got %v", err)
	}
}

func TestRouterHEADImplicit(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("home", "/", h)
	req := httptest.NewRequest("HEAD", "/", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatalf("HEAD should match GET route: %v", err)
	}
	if m.Name != "home" {
		t.Errorf("expected home, got %s", m.Name)
	}
}

func TestRouterHEADImplicitDisabled(t *testing.T) {
	r := New(WithImplicitHEAD(false))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("home", "/", h)
	req := httptest.NewRequest("HEAD", "/", nil)
	_, err := r.Match(req)
	if err != ErrMethodNotAllowed {
		t.Errorf("expected ErrMethodNotAllowed when implicit HEAD disabled, got %v", err)
	}
}

func TestRouterURLGeneration(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", h)

	u, err := r.URL("users.show", Params{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/users/42" {
		t.Errorf("expected /users/42, got %s", u.Path)
	}
}

func TestRouterPathGeneration(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("search", "/search{?q}", h)

	p, err := r.Path("search", Params{"q": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if p != "/search?q=hello" {
		t.Errorf("expected /search?q=hello, got %s", p)
	}
}

func TestRouterURLUnknownRoute(t *testing.T) {
	r := New()
	_, err := r.URL("nonexistent", nil)
	if err != ErrUnknownRoute {
		t.Errorf("expected ErrUnknownRoute, got %v", err)
	}
}

func TestContextHelpers(t *testing.T) {
	m := &Match{
		Name:   "test",
		Params: Params{"id": "42"},
	}
	ctx := withMatch(context.Background(), m)

	got, ok := MatchFromContext(ctx)
	if !ok || got != m {
		t.Error("MatchFromContext failed")
	}

	name, ok := RouteNameFromContext(ctx)
	if !ok || name != "test" {
		t.Error("RouteNameFromContext failed")
	}

	params, ok := ParamsFromContext(ctx)
	if !ok || params.Get("id") != "42" {
		t.Error("ParamsFromContext failed")
	}

	_, ok = MatchFromContext(context.Background())
	if ok {
		t.Error("should return false for empty context")
	}
}

func TestServeHTTPDispatch(t *testing.T) {
	r := New()
	r.GET("home", "/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, ok := MatchFromContext(req.Context())
		if !ok {
			t.Error("no match in context")
		}
		if m.Name != "home" {
			t.Errorf("expected home, got %s", m.Name)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestServeHTTPMethodNotAllowed(t *testing.T) {
	r := New()
	r.GET("home", "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestConstraintInt(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("users.show", "/users/{id}", h, WithConstraint(Int("id")))

	req := httptest.NewRequest("GET", "/users/42", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Params.Get("id") != "42" {
		t.Error("should match numeric id")
	}

	req = httptest.NewRequest("GET", "/users/abc", nil)
	_, err = r.Match(req)
	if err == nil {
		t.Error("should not match non-numeric id")
	}
}

func TestRouteIntrospection(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("home", "/", h)
	r.POST("create", "/items", h)

	rt, ok := r.Route("home")
	if !ok || rt.Name != "home" {
		t.Error("Route lookup failed")
	}
	_, ok = r.Route("nonexistent")
	if ok {
		t.Error("should not find nonexistent route")
	}

	routes := r.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Name != "home" || routes[1].Name != "create" {
		t.Error("routes order incorrect")
	}
}

func TestScopeBasic(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	scope := r.WithScope(
		WithNamePrefix("admin"),
		WithTemplatePrefix("/admin"),
	)
	if err := scope.GET("users", "/users", h); err != nil {
		t.Fatal(err)
	}

	rt, ok := r.Route("admin.users")
	if !ok {
		t.Fatal("expected admin.users route")
	}
	if rt.Template.String() != "/admin/users" {
		t.Errorf("expected /admin/users template, got %s", rt.Template.String())
	}
}

func TestScopeNested(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	r.Scope(func(s *Scope) {
		WithNamePrefix("api")(s)
		WithTemplatePrefix("/api")(s)
		s.Scope(func(inner *Scope) {
			WithNamePrefix("api.v1")(inner)
			WithTemplatePrefix("/api/v1")(inner)
			inner.GET("users", "/users", h)
		})
	})

	rt, ok := r.Route("api.v1.users")
	if !ok {
		t.Fatal("expected api.v1.users route")
	}
	if rt.Template.String() != "/api/v1/users" {
		t.Errorf("expected /api/v1/users, got %s", rt.Template.String())
	}
}

func TestScopeDefaults(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	scope := r.WithScope(WithScopeDefaults(Params{"format": "json"}))
	scope.GET("test", "/test", h)

	rt, ok := r.Route("test")
	if !ok {
		t.Fatal("expected test route")
	}
	if rt.Defaults.Get("format") != "json" {
		t.Error("scope defaults not applied")
	}
}

func TestScopeConstraints(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	scope := r.WithScope(WithScopeConstraint(Host("example.com")))
	scope.GET("test", "/test", h)

	rt, ok := r.Route("test")
	if !ok {
		t.Fatal("expected test route")
	}
	if len(rt.Constraints) != 1 {
		t.Errorf("expected 1 constraint from scope, got %d", len(rt.Constraints))
	}
}

func TestCanonicalRedirect(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.GET("search", "/search{?q}", h,
		WithCanonicalPolicy(CanonicalRedirect),
		WithRedirectCode(http.StatusMovedPermanently),
	)

	// Canonical request should dispatch normally
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/search?q=hello", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("canonical request: expected 200, got %d", w.Code)
	}
}

func TestUniqueRouteNames(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("test", "/a", h)
	err := r.GET("test", "/b", h)
	if !errors.Is(err, ErrDuplicateRoute) {
		t.Errorf("expected ErrDuplicateRoute, got %v", err)
	}
}

func TestDeterministicDispatch(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("a", "/test", h, WithPriority(1))
	r.GET("b", "/test", h, WithPriority(2))

	req := httptest.NewRequest("GET", "/test", nil)
	m1, _ := r.Match(req)
	m2, _ := r.Match(req)
	if m1.Name != m2.Name {
		t.Error("dispatch should be deterministic")
	}
	// Higher priority should win
	if m1.Name != "b" {
		t.Errorf("expected b (higher priority), got %s", m1.Name)
	}
}

func TestDefaultsNeverOverrideExtracted(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("test", "/users/{id}", h, WithDefaults(Params{"id": "default"}))

	req := httptest.NewRequest("GET", "/users/42", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Params.Get("id") != "42" {
		t.Errorf("default should not override extracted: got %s", m.Params.Get("id"))
	}
}

func TestConstraintsNeverMutateParams(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.GET("test", "/users/{id}", h, WithConstraint(Custom(func(rc *RequestContext, p Params) bool {
		// Constraint should not mutate params, but even if it does, the
		// original params should be preserved
		return true
	})))

	req := httptest.NewRequest("GET", "/users/42", nil)
	m, err := r.Match(req)
	if err != nil {
		t.Fatal(err)
	}
	if m.Params.Get("id") != "42" {
		t.Error("params should be unchanged after constraint")
	}
}

func TestContextReflectsSelectedRoute(t *testing.T) {
	r := New()
	r.GET("users.show", "/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, ok := MatchFromContext(req.Context())
		if !ok {
			t.Error("no match in context")
			return
		}
		if m.Name != "users.show" {
			t.Errorf("context route name: expected users.show, got %s", m.Name)
		}
		if m.Params.Get("id") != "42" {
			t.Errorf("context params: expected id=42, got %s", m.Params.Get("id"))
		}
		name, _ := RouteNameFromContext(req.Context())
		if name != "users.show" {
			t.Errorf("RouteNameFromContext: expected users.show, got %s", name)
		}
		params, _ := ParamsFromContext(req.Context())
		if params.Get("id") != "42" {
			t.Errorf("ParamsFromContext: expected id=42, got %s", params.Get("id"))
		}
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(w, req)
}
