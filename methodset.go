package dispatch

import (
	"fmt"
	"net/http"
	"strings"
)

// MethodSet represents a set of HTTP methods allowed for a route. Each method
// is encoded as a single bit in a uint16 value so that sets can be combined
// with bitwise OR.
type MethodSet uint16

const (
	// GET represents the HTTP GET method.
	GET MethodSet = 1 << iota
	// HEAD represents the HTTP HEAD method.
	HEAD
	// POST represents the HTTP POST method.
	POST
	// PUT represents the HTTP PUT method.
	PUT
	// PATCH represents the HTTP PATCH method.
	PATCH
	// DELETE represents the HTTP DELETE method.
	DELETE
	// OPTIONS represents the HTTP OPTIONS method.
	OPTIONS
	// TRACE represents the HTTP TRACE method.
	TRACE
	// CONNECT represents the HTTP CONNECT method.
	CONNECT
)

// Has reports whether ms includes every method in other.
func (ms MethodSet) Has(other MethodSet) bool {
	return ms&other == other
}

// String returns a pipe-separated list of the HTTP methods in the set,
// for example "GET|HEAD". An empty set returns "<none>".
func (ms MethodSet) String() string {
	if ms == 0 {
		return "<none>"
	}
	methods := []struct {
		bit  MethodSet
		name string
	}{
		{GET, "GET"},
		{HEAD, "HEAD"},
		{POST, "POST"},
		{PUT, "PUT"},
		{PATCH, "PATCH"},
		{DELETE, "DELETE"},
		{OPTIONS, "OPTIONS"},
		{TRACE, "TRACE"},
		{CONNECT, "CONNECT"},
	}
	var parts []string
	for _, m := range methods {
		if ms&m.bit != 0 {
			parts = append(parts, m.name)
		}
	}
	if len(parts) == 0 {
		return "<none>"
	}
	return strings.Join(parts, "|")
}

// MethodSetFrom converts HTTP method name strings to a MethodSet, returning an
// error for unrecognised method names. This is useful in route registration
// helpers that accept string method arguments.
func MethodSetFrom(methods ...string) (MethodSet, error) {
	var ms MethodSet
	for _, name := range methods {
		bit := methodFromString(strings.ToUpper(name))
		if bit == 0 {
			return 0, &MethodError{Method: name}
		}
		ms |= bit
	}
	return ms, nil
}

// MethodError is returned by MethodSetFrom when an unrecognised HTTP method
// name is encountered.
type MethodError struct {
	// Method is the unrecognised method name.
	Method string
}

// Error returns a human-readable description of the unrecognised method.
func (e *MethodError) Error() string {
	return "unrecognised HTTP method: " + e.Method
}

// MethodFromString returns the MethodSet bit for a standard HTTP method string.
// It returns 0 and an error if the method is not one of the nine standard methods.
// It is case-sensitive and rejects lowercase or mixed-case method strings.
func MethodFromString(method string) (MethodSet, error) {
	m := methodFromString(method)
	if m == 0 {
		return 0, fmt.Errorf("dispatch: unknown HTTP method %q", method)
	}
	return m, nil
}

// methodFromString maps an HTTP method string to its MethodSet bit.
// Returns 0 for unknown methods.
func methodFromString(m string) MethodSet {
	switch m {
	case http.MethodGet:
		return GET
	case http.MethodHead:
		return HEAD
	case http.MethodPost:
		return POST
	case http.MethodPut:
		return PUT
	case http.MethodPatch:
		return PATCH
	case http.MethodDelete:
		return DELETE
	case http.MethodOptions:
		return OPTIONS
	case http.MethodTrace:
		return TRACE
	case http.MethodConnect:
		return CONNECT
	default:
		return 0
	}
}
