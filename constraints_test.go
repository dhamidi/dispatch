package dispatch

import (
	"regexp"
	"testing"
)

func TestExact_Match(t *testing.T) {
	c := Exact("format", "json")
	if !c.Check(nil, Params{"format": "json"}) {
		t.Error("expected match")
	}
}

func TestExact_NoMatch(t *testing.T) {
	c := Exact("format", "json")
	if c.Check(nil, Params{"format": "xml"}) {
		t.Error("expected no match")
	}
}

func TestExact_MissingKey(t *testing.T) {
	c := Exact("format", "json")
	if c.Check(nil, Params{}) {
		t.Error("expected no match for missing key")
	}
}

func TestOneOf_Match(t *testing.T) {
	c := OneOf("format", "json", "xml", "csv")
	if !c.Check(nil, Params{"format": "xml"}) {
		t.Error("expected match")
	}
}

func TestOneOf_NoMatch(t *testing.T) {
	c := OneOf("format", "json", "xml")
	if c.Check(nil, Params{"format": "csv"}) {
		t.Error("expected no match")
	}
}

func TestOneOf_EmptyValues(t *testing.T) {
	c := OneOf("format")
	if c.Check(nil, Params{"format": "json"}) {
		t.Error("expected no match with empty value set")
	}
}

func TestOneOf_MissingKey(t *testing.T) {
	c := OneOf("format", "json")
	if c.Check(nil, Params{}) {
		t.Error("expected no match for missing key")
	}
}

func TestRegexp_FullMatch(t *testing.T) {
	re := regexp.MustCompile(`^[a-z0-9-]+$`)
	c := Regexp("slug", re)
	if !c.Check(nil, Params{"slug": "hello-world-42"}) {
		t.Error("expected match")
	}
}

func TestRegexp_PartialMatch(t *testing.T) {
	// This regex is already anchored, so partial should fail
	re := regexp.MustCompile(`^[a-z]+$`)
	c := Regexp("slug", re)
	if c.Check(nil, Params{"slug": "Hello"}) {
		t.Error("expected no match for partial/case mismatch")
	}
}

func TestRegexp_NoMatch(t *testing.T) {
	re := regexp.MustCompile(`^\d+$`)
	c := Regexp("id", re)
	if c.Check(nil, Params{"id": "abc"}) {
		t.Error("expected no match")
	}
}

func TestMethods_SingleMethod(t *testing.T) {
	c := Methods(GET)
	rc := &RequestContext{Method: "GET"}
	if !c.Check(rc, nil) {
		t.Error("expected match for GET")
	}
	rc2 := &RequestContext{Method: "POST"}
	if c.Check(rc2, nil) {
		t.Error("expected no match for POST")
	}
}

func TestMethods_CombinedMethods(t *testing.T) {
	c := Methods(GET | HEAD)
	rc := &RequestContext{Method: "HEAD"}
	if !c.Check(rc, nil) {
		t.Error("expected match for HEAD")
	}
	rc2 := &RequestContext{Method: "DELETE"}
	if c.Check(rc2, nil) {
		t.Error("expected no match for DELETE")
	}
}

func TestHost_Match(t *testing.T) {
	c := Host("api.example.com")
	rc := &RequestContext{Host: "api.example.com"}
	if !c.Check(rc, nil) {
		t.Error("expected match")
	}
}

func TestHost_CaseInsensitive(t *testing.T) {
	c := Host("API.Example.COM")
	rc := &RequestContext{Host: "api.example.com"}
	if !c.Check(rc, nil) {
		t.Error("expected case-insensitive match")
	}
}

func TestHost_NoMatch(t *testing.T) {
	c := Host("api.example.com")
	rc := &RequestContext{Host: "www.example.com"}
	if c.Check(rc, nil) {
		t.Error("expected no match")
	}
}

func TestCustom_Constraint(t *testing.T) {
	c := Custom(func(rc *RequestContext, p Params) bool {
		return p["version"] != ""
	})
	if !c.Check(nil, Params{"version": "v1"}) {
		t.Error("expected match")
	}
	if c.Check(nil, Params{}) {
		t.Error("expected no match")
	}
}
