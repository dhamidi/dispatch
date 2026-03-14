package dispatch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestIsCanonicalURL_Matching(t *testing.T) {
	a := &url.URL{Path: "/users/42"}
	b := &url.URL{Path: "/users/42"}
	if !isCanonicalURL(a, b) {
		t.Error("identical URLs should be canonical")
	}
}

func TestIsCanonicalURL_DifferentPath(t *testing.T) {
	a := &url.URL{Path: "/users/42"}
	b := &url.URL{Path: "/users/43"}
	if isCanonicalURL(a, b) {
		t.Error("different paths should not be canonical")
	}
}

func TestIsCanonicalURL_QueryOrder(t *testing.T) {
	a := &url.URL{Path: "/search", RawQuery: "b=2&a=1"}
	b := &url.URL{Path: "/search", RawQuery: "a=1&b=2"}
	if !isCanonicalURL(a, b) {
		t.Error("URLs with same query params in different order should be canonical")
	}
}

func TestNormalizeQuery_Empty(t *testing.T) {
	if got := normalizeQuery(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestNormalizeQuery_Sorted(t *testing.T) {
	got := normalizeQuery("z=1&a=2")
	if got != "a=2&z=1" {
		t.Errorf("expected a=2&z=1, got %q", got)
	}
}

func TestCanonicalRedirect_WithCustomRedirectCode(t *testing.T) {
	r := New()
	r.GET("search", "/search{?q}", noopHandler,
		WithCanonicalPolicy(CanonicalRedirect),
		WithRedirectCode(http.StatusTemporaryRedirect),
	)

	req := httptest.NewRequest("GET", "/search?q=hi&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307, got %d", rec.Code)
	}
}

func TestCanonicalAnnotate_IsCanonicalTrue(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, ok := MatchFromContext(req.Context())
		if !ok {
			t.Fatal("no match")
		}
		if !m.IsCanonical {
			t.Error("expected IsCanonical=true for canonical URL")
		}
	})

	r.GET("items.show", "/items/{id}", h, WithCanonicalPolicy(CanonicalAnnotate))

	req := httptest.NewRequest("GET", "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
