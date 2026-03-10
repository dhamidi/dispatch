// Package dispatch provides semantic HTTP routing for Go.
//
// It is inspired by Rails ActionDispatch while remaining idiomatic to Go
// and compatible with the standard net/http package.
//
// Routing is built on URI templates (github.com/dhamidi/uritemplate) as the
// single source of truth for both inbound matching and outbound URL generation.
package dispatch

import (
	"net/url"
)

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
