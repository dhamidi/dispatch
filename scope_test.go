package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhamidi/uritemplate"
)

func TestScope_Handle(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Scope(func(s *Scope) {
		tmpl := uritemplate.MustParse("/admin/dashboard")
		err := s.Handle(Route{
			Name:     "dashboard",
			Methods:  GET,
			Template: tmpl,
			Handler:  h,
		})
		if err != nil {
			t.Fatal(err)
		}
	}, WithNamePrefix("admin"))

	req := httptest.NewRequest("GET", "/admin/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestScope_POST(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r.Scope(func(s *Scope) {
		if err := s.POST("create", "/items", h); err != nil {
			t.Fatal(err)
		}
	})

	req := httptest.NewRequest("POST", "/items", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("POST handler not called")
	}
}

func TestScope_PUT(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r.Scope(func(s *Scope) {
		if err := s.PUT("update", "/items/{id}", h); err != nil {
			t.Fatal(err)
		}
	})

	req := httptest.NewRequest("PUT", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("PUT handler not called")
	}
}

func TestScope_PATCH(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r.Scope(func(s *Scope) {
		if err := s.PATCH("patch", "/items/{id}", h); err != nil {
			t.Fatal(err)
		}
	})

	req := httptest.NewRequest("PATCH", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("PATCH handler not called")
	}
}

func TestScope_DELETE(t *testing.T) {
	r := New()
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r.Scope(func(s *Scope) {
		if err := s.DELETE("destroy", "/items/{id}", h); err != nil {
			t.Fatal(err)
		}
	})

	req := httptest.NewRequest("DELETE", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Error("DELETE handler not called")
	}
}

func TestWithScopeQueryMode(t *testing.T) {
	r := New()
	r.Scope(func(s *Scope) {
		s.GET("foo", "/foo", noopHandler)
	}, WithScopeQueryMode(QueryStrict))

	req := httptest.NewRequest("GET", "/foo?extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for strict query mode, got %d", rec.Code)
	}
}

func TestWithScopeCanonicalPolicy(t *testing.T) {
	r := New()
	r.Scope(func(s *Scope) {
		s.GET("search", "/search{?q}", noopHandler)
	}, WithScopeCanonicalPolicy(CanonicalRedirect))

	req := httptest.NewRequest("GET", "/search?q=hi&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", rec.Code)
	}
}

func TestWithScopeMetadata(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, ok := MatchFromContext(req.Context())
		if !ok {
			t.Fatal("no match")
		}
		if m.Route.Metadata["env"] != "staging" {
			t.Errorf("expected env=staging, got %q", m.Route.Metadata["env"])
		}
	})

	r.Scope(func(s *Scope) {
		s.GET("foo", "/foo", h)
	}, WithScopeMetadata("env", "staging"))

	req := httptest.NewRequest("GET", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}

func TestScope_NestedMetadata_InnerOverridesOuter(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, _ := MatchFromContext(req.Context())
		if m.Route.Metadata["env"] != "prod" {
			t.Errorf("expected inner env=prod, got %q", m.Route.Metadata["env"])
		}
		if m.Route.Metadata["region"] != "us" {
			t.Errorf("expected inherited region=us, got %q", m.Route.Metadata["region"])
		}
	})

	r.Scope(func(outer *Scope) {
		outer.Scope(func(inner *Scope) {
			inner.GET("item", "/items/{id}", h)
		}, WithScopeMetadata("env", "prod"))
	}, WithScopeMetadata("env", "staging"), WithScopeMetadata("region", "us"))

	req := httptest.NewRequest("GET", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}

func TestScope_NestedConstraints_OuterFirst(t *testing.T) {
	r := New()
	var order []string
	c1 := Custom(func(rc *RequestContext, p Params) bool {
		order = append(order, "outer")
		return true
	})
	c2 := Custom(func(rc *RequestContext, p Params) bool {
		order = append(order, "inner")
		return true
	})

	r.Scope(func(outer *Scope) {
		outer.Scope(func(inner *Scope) {
			inner.GET("item", "/items/{id}", noopHandler)
		}, WithScopeConstraint(c2))
	}, WithScopeConstraint(c1))

	req := httptest.NewRequest("GET", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(order) < 2 || order[0] != "outer" || order[1] != "inner" {
		t.Errorf("expected [outer, inner], got %v", order)
	}
}

func TestScope_NestedDefaults_InnerOverridesOuter(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p, _ := ParamsFromContext(req.Context())
		if p.Get("format") != "xml" {
			t.Errorf("expected inner default format=xml, got %q", p.Get("format"))
		}
		if p.Get("version") != "v1" {
			t.Errorf("expected inherited default version=v1, got %q", p.Get("version"))
		}
	})

	r.Scope(func(outer *Scope) {
		outer.Scope(func(inner *Scope) {
			inner.GET("item", "/items/{id}", h)
		}, WithScopeDefaults(Params{"format": "xml"}))
	}, WithScopeDefaults(Params{"format": "json", "version": "v1"}))

	req := httptest.NewRequest("GET", "/items/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}
