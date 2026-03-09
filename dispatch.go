// Package dispatch provides semantic HTTP routing for Go.
//
// It is inspired by Rails ActionDispatch while remaining idiomatic to Go
// and compatible with the standard net/http package.
//
// Routing is built on URI templates (github.com/dhamidi/uritemplate) as the
// single source of truth for both inbound matching and outbound URL generation.
package dispatch

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/dhamidi/uritemplate"
)

// Params represents route parameters after extraction, default application,
// and normalization. Keys are case-sensitive; values are strings.
// Callers should treat a Params returned in a Match as read-only.
type Params map[string]string

// Get returns the value for key, or an empty string if not present.
func (p Params) Get(key string) string {
	return p[key]
}

// Lookup returns the value for key and a boolean indicating whether it was found.
func (p Params) Lookup(key string) (string, bool) {
	v, ok := p[key]
	return v, ok
}

// Clone returns a shallow copy of p.
func (p Params) Clone() Params {
	if p == nil {
		return nil
	}
	c := make(Params, len(p))
	for k, v := range p {
		c[k] = v
	}
	return c
}

// MethodSet is a bitmask of allowed HTTP methods.
type MethodSet uint16

const (
	MethodGET     MethodSet = 1 << iota // GET
	MethodHEAD                          // HEAD
	MethodPOST                          // POST
	MethodPUT                           // PUT
	MethodPATCH                         // PATCH
	MethodDELETE                        // DELETE
	MethodOPTIONS                       // OPTIONS
	MethodTRACE                         // TRACE
	MethodCONNECT                       // CONNECT
)

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
		return MethodGET
	case http.MethodHead:
		return MethodHEAD
	case http.MethodPost:
		return MethodPOST
	case http.MethodPut:
		return MethodPUT
	case http.MethodPatch:
		return MethodPATCH
	case http.MethodDelete:
		return MethodDELETE
	case http.MethodOptions:
		return MethodOPTIONS
	case http.MethodTrace:
		return MethodTRACE
	case http.MethodConnect:
		return MethodCONNECT
	default:
		return 0
	}
}

// contains reports whether ms includes m.
func (ms MethodSet) contains(m MethodSet) bool {
	return ms&m != 0
}

// QueryMode controls how template query variables participate in matching.
type QueryMode uint8

const (
	// QueryLoose: undeclared query params are ignored for match eligibility.
	QueryLoose QueryMode = iota
	// QueryCanonical: undeclared params don't prevent matching but may be
	// dropped from the canonical URL.
	QueryCanonical
	// QueryStrict: undeclared query params reject the candidate route.
	QueryStrict
)

// CanonicalPolicy controls what happens when the request URL differs from the
// canonical URL computed for the matched route.
type CanonicalPolicy uint8

const (
	// CanonicalIgnore: no canonical comparison is performed.
	CanonicalIgnore CanonicalPolicy = iota
	// CanonicalAnnotate: canonical URL is computed and exposed in Match;
	// no automatic redirect.
	CanonicalAnnotate
	// CanonicalRedirect: router emits a redirect when the URL is non-canonical.
	CanonicalRedirect
	// CanonicalReject: non-canonical requests are rejected (not dispatchable).
	CanonicalReject
)

// RequestContext holds request attributes used during routing and constraint
// evaluation.
type RequestContext struct {
	Request *http.Request
	URL     *url.URL
	Method  string
	Host    string
}

// Constraint refines a candidate route after URI template extraction.
// Implementations must be side-effect free and must not mutate Params.
type Constraint interface {
	Check(*RequestContext, Params) bool
}

// ConstraintFunc is a function adapter that implements Constraint.
type ConstraintFunc func(*RequestContext, Params) bool

// Check implements Constraint.
func (f ConstraintFunc) Check(rc *RequestContext, p Params) bool {
	return f(rc, p)
}

// Route defines a semantic endpoint.
type Route struct {
	// Name is the stable application-defined identifier. Must be non-empty
	// and unique within a router instance.
	Name string

	// Methods is the set of allowed HTTP methods. Must not be zero.
	Methods MethodSet

	// Template is the parsed URI template used for matching and URL generation.
	// Must not be nil.
	Template *uritemplate.Template

	// Handler is the http.Handler invoked when this route is selected.
	// Must not be nil for dispatchable routes.
	Handler http.Handler

	// Defaults provides fallback values for template variables not present in
	// the request. Applied after extraction; never overrides extracted values.
	Defaults Params

	// Constraints are post-match refinement rules evaluated in order.
	Constraints []Constraint

	// QueryMode controls undeclared query parameter behavior.
	// Defaults to QueryLoose if unset.
	QueryMode QueryMode

	// CanonicalPolicy controls canonical URL enforcement behavior.
	// Defaults to CanonicalIgnore if unset.
	CanonicalPolicy CanonicalPolicy

	// RedirectCode is the HTTP status code used for canonical redirects.
	// Should default to http.StatusMovedPermanently (301) if zero and
	// CanonicalRedirect is active.
	RedirectCode int

	// Priority is an explicit tie-breaker. Higher values win.
	Priority int

	// Metadata is an optional application-defined string map treated as
	// opaque by the router.
	Metadata map[string]string
}

// candidateScore holds the scoring breakdown used for deterministic
// multi-candidate selection (§10.7.2).
type candidateScore struct {
	LiteralSegments int // more is better
	ConstrainedVars int // more is better
	BroadVars       int // fewer is better
	QueryMatches    int // more is better
	Priority        int // more is better
	Registration    int // lower index is better
}

// less reports whether s is a worse match than other.
// Used to pick the best candidate: returns true if s should lose to other.
func (s candidateScore) less(other candidateScore) bool {
	if s.LiteralSegments != other.LiteralSegments {
		return s.LiteralSegments < other.LiteralSegments
	}
	if s.ConstrainedVars != other.ConstrainedVars {
		return s.ConstrainedVars < other.ConstrainedVars
	}
	if s.BroadVars != other.BroadVars {
		return s.BroadVars > other.BroadVars
	}
	if s.QueryMatches != other.QueryMatches {
		return s.QueryMatches < other.QueryMatches
	}
	if s.Priority != other.Priority {
		return s.Priority < other.Priority
	}
	return s.Registration > other.Registration
}

// Match represents the result of resolving a request to a route.
type Match struct {
	// Route is the selected route definition.
	Route *Route

	// Name equals Route.Name.
	Name string

	// Params contains extracted values merged with defaults.
	Params Params

	// Method is the HTTP method of the matched request.
	Method string

	// CanonicalURL is the normalized URL computed from the matched route.
	// May be nil when canonical computation is disabled.
	CanonicalURL *url.URL

	// IsCanonical indicates whether the request URL equals the canonical URL.
	IsCanonical bool

	// RedirectNeeded indicates that canonical redirect policy requires a
	// redirect instead of normal handler dispatch.
	RedirectNeeded bool

	// score is the internal scoring used during candidate selection.
	score candidateScore
}
