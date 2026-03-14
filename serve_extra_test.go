package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDispatch_CanonicalRedirect_DefaultCode(t *testing.T) {
	r := New(WithDefaultCanonicalPolicy(CanonicalRedirect))
	r.GET("search", "/search{?q}", noopHandler)

	// Request with extra params triggers canonical redirect with default 301
	req := httptest.NewRequest("GET", "/search?q=hello&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", rec.Code)
	}
}

func TestDispatch_CanonicalRedirect_RouteRedirectCode(t *testing.T) {
	r := New()
	r.GET("search", "/search{?q}", noopHandler,
		WithCanonicalPolicy(CanonicalRedirect),
		WithRedirectCode(http.StatusFound),
	)

	req := httptest.NewRequest("GET", "/search?q=hello&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
}

func TestDispatch_MethodNotAllowed(t *testing.T) {
	r := New()
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("DELETE", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestDispatch_NotFound(t *testing.T) {
	r := New()
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("GET", "/bar", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestSlashRedirect_AddSlash(t *testing.T) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.GET("admin", "/admin/", noopHandler)

	req := httptest.NewRequest("GET", "/admin", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected redirect to /admin/, got %d", rec.Code)
	}
}

func TestSlashRedirect_RemoveSlash(t *testing.T) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.GET("admin", "/admin", noopHandler)

	req := httptest.NewRequest("GET", "/admin/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected redirect to /admin, got %d", rec.Code)
	}
}

func TestSlashRedirect_PreservesQueryString(t *testing.T) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("GET", "/foo/?key=val", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/foo?key=val" {
		t.Errorf("expected /foo?key=val, got %q", loc)
	}
}

func TestSlashRedirect_NoMatch(t *testing.T) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("GET", "/bar", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
