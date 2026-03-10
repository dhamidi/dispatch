package dispatch

import (
	"errors"
	"testing"
)

func TestURL_UnknownRoute(t *testing.T) {
	r := New()
	_, err := r.URL("nonexistent", nil)
	if !errors.Is(err, ErrUnknownRoute) {
		t.Errorf("expected ErrUnknownRoute, got %v", err)
	}
}

func TestURL_Success(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler); err != nil {
		t.Fatalf("register: %v", err)
	}
	u, err := r.URL("users.show", Params{"id": "99"})
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if u.Path != "/users/99" {
		t.Errorf("expected path /users/99, got %s", u.Path)
	}
}

func TestURL_MissingParam(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler); err != nil {
		t.Fatalf("register: %v", err)
	}
	u, err := r.URL("users.show", nil)
	if err != nil {
		// uritemplate returned an error for missing required variable.
		if !errors.Is(err, ErrMissingParam) {
			t.Errorf("expected ErrMissingParam, got %v", err)
		}
		return
	}
	// uritemplate silently omits the variable — document the behavior.
	// When a required path variable is absent, uritemplate may produce
	// a path with the variable simply removed (e.g. "/users/").
	t.Logf("uritemplate silently omitted missing variable; got path %q", u.Path)
	if u.Path != "/users/" {
		t.Errorf("expected path /users/ for missing param, got %s", u.Path)
	}
}

func TestURL_DefaultsMerged(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler, WithDefaults(Params{"id": "1"})); err != nil {
		t.Fatalf("register: %v", err)
	}
	u, err := r.URL("users.show", nil)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if u.Path != "/users/1" {
		t.Errorf("expected path /users/1, got %s", u.Path)
	}
}

func TestURL_CallerOverridesDefaults(t *testing.T) {
	r := New()
	if err := r.GET("users.show", "/users/{id}", noopHandler, WithDefaults(Params{"id": "1"})); err != nil {
		t.Fatalf("register: %v", err)
	}
	u, err := r.URL("users.show", Params{"id": "99"})
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if u.Path != "/users/99" {
		t.Errorf("expected path /users/99, got %s", u.Path)
	}
}

func TestPath_Success(t *testing.T) {
	r := New()
	if err := r.GET("search", "/search{?q}", noopHandler); err != nil {
		t.Fatalf("register: %v", err)
	}
	path, err := r.Path("search", Params{"q": "hello"})
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if !containsSubstring(path, "q=hello") {
		t.Errorf("expected path to contain q=hello, got %s", path)
	}
}

func TestPath_UnknownRoute(t *testing.T) {
	r := New()
	_, err := r.Path("nope", nil)
	if !errors.Is(err, ErrUnknownRoute) {
		t.Errorf("expected ErrUnknownRoute, got %v", err)
	}
}

// containsSubstring reports whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
