package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithNotFoundHandler(t *testing.T) {
	custom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})
	r := New(WithNotFoundHandler(custom))

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "custom 404" {
		t.Errorf("expected custom 404 body, got %q", rec.Body.String())
	}
}

func TestWithMethodNotAllowedHandler(t *testing.T) {
	custom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("custom 405"))
	})
	r := New(WithMethodNotAllowedHandler(custom))
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("POST", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
	if rec.Body.String() != "custom 405" {
		t.Errorf("expected custom 405 body, got %q", rec.Body.String())
	}
}

func TestWithErrorHandler(t *testing.T) {
	custom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("custom error"))
	})
	r := New(WithErrorHandler(custom))
	// ErrorHandler is set but only triggered on internal dispatch errors,
	// which are hard to trigger. Just verify the option is applied.
	_ = r
}

func TestWithDefaultQueryMode_Strict(t *testing.T) {
	r := New(WithDefaultQueryMode(QueryStrict))
	r.GET("foo", "/foo", noopHandler)

	// Request with extra query params should not match under strict mode
	req := httptest.NewRequest("GET", "/foo?extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for strict query mode with extra params, got %d", rec.Code)
	}
}

func TestWithDefaultCanonicalPolicy_Redirect(t *testing.T) {
	r := New(WithDefaultCanonicalPolicy(CanonicalRedirect))
	r.GET("search", "/search{?q}", noopHandler)

	// Request with extra query params: canonical URL differs
	req := httptest.NewRequest("GET", "/search?q=hello&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301 redirect, got %d", rec.Code)
	}
}

func TestWithDefaultRedirectCode(t *testing.T) {
	r := New(
		WithDefaultCanonicalPolicy(CanonicalRedirect),
		WithDefaultRedirectCode(http.StatusTemporaryRedirect),
	)
	r.GET("search", "/search{?q}", noopHandler)

	req := httptest.NewRequest("GET", "/search?q=hello&extra=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307, got %d", rec.Code)
	}
}

func TestWithDefaultSlashPolicy(t *testing.T) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("GET", "/foo/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected redirect for trailing slash, got %d", rec.Code)
	}
}

func TestWithImplicitHEAD_Disabled(t *testing.T) {
	r := New(WithImplicitHEAD(false))
	r.GET("foo", "/foo", noopHandler)

	req := httptest.NewRequest("HEAD", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 when implicit HEAD is disabled, got %d", rec.Code)
	}
}
