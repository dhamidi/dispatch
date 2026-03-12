package dispatch

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// requestWithParams creates an *http.Request whose context contains a Match
// with the given params.
func requestWithParams(params Params) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	m := &Match{Params: params}
	ctx := storeMatchInContext(r.Context(), m)
	return r.WithContext(ctx)
}

// requestWithoutMatch creates an *http.Request with no dispatch Match in context.
func requestWithoutMatch() *http.Request {
	return httptest.NewRequest("GET", "/", nil)
}

func TestParamString(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		r := requestWithParams(Params{"slug": "hello-world"})
		v, ok := ParamString(r, "slug")
		if !ok || v != "hello-world" {
			t.Errorf("ParamString = (%q, %v), want (%q, true)", v, ok, "hello-world")
		}
	})
	t.Run("present empty value", func(t *testing.T) {
		r := requestWithParams(Params{"slug": ""})
		v, ok := ParamString(r, "slug")
		if !ok || v != "" {
			t.Errorf("ParamString = (%q, %v), want (%q, true)", v, ok, "")
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := requestWithParams(Params{"other": "x"})
		v, ok := ParamString(r, "slug")
		if ok || v != "" {
			t.Errorf("ParamString = (%q, %v), want (%q, false)", v, ok, "")
		}
	})
	t.Run("no match context", func(t *testing.T) {
		r := requestWithoutMatch()
		v, ok := ParamString(r, "slug")
		if ok || v != "" {
			t.Errorf("ParamString = (%q, %v), want (%q, false)", v, ok, "")
		}
	})
}

func TestParamInt(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := requestWithParams(Params{"id": "42"})
		v, err := ParamInt(r, "id")
		if err != nil || v != 42 {
			t.Errorf("ParamInt = (%d, %v), want (42, nil)", v, err)
		}
	})
	t.Run("negative", func(t *testing.T) {
		r := requestWithParams(Params{"id": "-7"})
		v, err := ParamInt(r, "id")
		if err != nil || v != -7 {
			t.Errorf("ParamInt = (%d, %v), want (-7, nil)", v, err)
		}
	})
	t.Run("parse error", func(t *testing.T) {
		r := requestWithParams(Params{"id": "abc"})
		_, err := ParamInt(r, "id")
		if err == nil {
			t.Fatal("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Name != "id" || pe.Raw != "abc" {
			t.Errorf("ParamError = {Name: %q, Raw: %q}, want {Name: %q, Raw: %q}", pe.Name, pe.Raw, "id", "abc")
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := requestWithParams(Params{})
		_, err := ParamInt(r, "id")
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
}

func TestParamInt64(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := requestWithParams(Params{"id": "9223372036854775807"})
		v, err := ParamInt64(r, "id")
		if err != nil || v != 9223372036854775807 {
			t.Errorf("ParamInt64 = (%d, %v), want (9223372036854775807, nil)", v, err)
		}
	})
	t.Run("parse error", func(t *testing.T) {
		r := requestWithParams(Params{"id": "abc"})
		_, err := ParamInt64(r, "id")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Errorf("expected strconv.ErrSyntax via Unwrap, got %v", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := requestWithParams(Params{})
		_, err := ParamInt64(r, "id")
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
	t.Run("no match context", func(t *testing.T) {
		r := requestWithoutMatch()
		_, err := ParamInt64(r, "id")
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
}

func TestParamFloat64(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := requestWithParams(Params{"lat": "3.14"})
		v, err := ParamFloat64(r, "lat")
		if err != nil || v != 3.14 {
			t.Errorf("ParamFloat64 = (%f, %v), want (3.14, nil)", v, err)
		}
	})
	t.Run("parse error", func(t *testing.T) {
		r := requestWithParams(Params{"lat": "not-a-float"})
		_, err := ParamFloat64(r, "lat")
		if err == nil {
			t.Fatal("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := requestWithParams(Params{})
		_, err := ParamFloat64(r, "lat")
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
}

func TestParamBool(t *testing.T) {
	trueCases := []string{"true", "TRUE", "True", "1", "yes", "YES", "Yes"}
	for _, tc := range trueCases {
		t.Run("true/"+tc, func(t *testing.T) {
			r := requestWithParams(Params{"flag": tc})
			v, err := ParamBool(r, "flag")
			if err != nil || !v {
				t.Errorf("ParamBool(%q) = (%v, %v), want (true, nil)", tc, v, err)
			}
		})
	}
	falseCases := []string{"false", "FALSE", "False", "0", "no", "NO", "No"}
	for _, tc := range falseCases {
		t.Run("false/"+tc, func(t *testing.T) {
			r := requestWithParams(Params{"flag": tc})
			v, err := ParamBool(r, "flag")
			if err != nil || v {
				t.Errorf("ParamBool(%q) = (%v, %v), want (false, nil)", tc, v, err)
			}
		})
	}
	t.Run("invalid", func(t *testing.T) {
		r := requestWithParams(Params{"flag": "maybe"})
		_, err := ParamBool(r, "flag")
		if err == nil {
			t.Fatal("expected error for invalid bool value")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := requestWithParams(Params{})
		_, err := ParamBool(r, "flag")
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
}

func TestMustParamInt64(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		r := requestWithParams(Params{"id": "99"})
		v := MustParamInt64(r, "id")
		if v != 99 {
			t.Errorf("MustParamInt64 = %d, want 99", v)
		}
	})
	t.Run("panics on missing", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Fatal("expected panic")
			}
		}()
		r := requestWithParams(Params{})
		MustParamInt64(r, "id")
	})
	t.Run("panics on invalid", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Fatal("expected panic")
			}
		}()
		r := requestWithParams(Params{"id": "abc"})
		MustParamInt64(r, "id")
	})
}

func TestMustParamInt(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		r := requestWithParams(Params{"id": "7"})
		v := MustParamInt(r, "id")
		if v != 7 {
			t.Errorf("MustParamInt = %d, want 7", v)
		}
	})
	t.Run("panics on missing", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Fatal("expected panic")
			}
		}()
		r := requestWithParams(Params{})
		MustParamInt(r, "id")
	})
}

// testParamValue is a custom ParamValue for testing ParamAs.
type testParamValue struct {
	parsed string
}

func (v *testParamValue) String() string {
	return v.parsed
}

func (v *testParamValue) Set(raw string) error {
	if raw == "bad" {
		return fmt.Errorf("invalid value")
	}
	v.parsed = "parsed:" + raw
	return nil
}

func TestParamAs(t *testing.T) {
	t.Run("valid custom type", func(t *testing.T) {
		r := requestWithParams(Params{"slug": "hello"})
		var v testParamValue
		err := ParamAs(r, "slug", &v)
		if err != nil {
			t.Fatalf("ParamAs returned error: %v", err)
		}
		if v.parsed != "parsed:hello" {
			t.Errorf("parsed = %q, want %q", v.parsed, "parsed:hello")
		}
	})
	t.Run("Set error", func(t *testing.T) {
		r := requestWithParams(Params{"slug": "bad"})
		var v testParamValue
		err := ParamAs(r, "slug", &v)
		if err == nil {
			t.Fatal("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Name != "slug" || pe.Raw != "bad" {
			t.Errorf("ParamError = {Name: %q, Raw: %q}, want {Name: %q, Raw: %q}", pe.Name, pe.Raw, "slug", "bad")
		}
	})
	t.Run("missing param", func(t *testing.T) {
		r := requestWithParams(Params{})
		var v testParamValue
		err := ParamAs(r, "slug", &v)
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
	t.Run("no match context", func(t *testing.T) {
		r := requestWithoutMatch()
		var v testParamValue
		err := ParamAs(r, "slug", &v)
		if !errors.Is(err, ErrParamNotFound) {
			t.Errorf("err = %v, want ErrParamNotFound", err)
		}
	})
}

// roundTripValue is a ParamValue that stores raw strings directly,
// enabling exact round-trip fidelity for testing.
type roundTripValue struct {
	raw string
}

func (v *roundTripValue) String() string { return v.raw }
func (v *roundTripValue) Set(raw string) error {
	v.raw = raw
	return nil
}

func TestParamValue_RoundTrip(t *testing.T) {
	var v roundTripValue
	if err := v.Set("hello-world"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	s := v.String()
	if s != "hello-world" {
		t.Fatalf("String() = %q, want %q", s, "hello-world")
	}

	// Re-parse the output of String() and verify we get the same result.
	var v2 roundTripValue
	if err := v2.Set(s); err != nil {
		t.Fatalf("Set(String()) returned error: %v", err)
	}
	if v2.String() != v.String() {
		t.Errorf("round-trip failed: Set(%q).String() = %q, want %q", s, v2.String(), v.String())
	}
}

func TestParamError_Unwrap(t *testing.T) {
	// Verify errors.Is works through ParamError wrapping.
	inner := strconv.ErrSyntax
	pe := &ParamError{Name: "id", Raw: "abc", Err: inner}
	if !errors.Is(pe, strconv.ErrSyntax) {
		t.Error("errors.Is(ParamError, strconv.ErrSyntax) should be true")
	}
}
