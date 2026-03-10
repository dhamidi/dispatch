package dispatch

import "net/url"

// candidateScore holds the precomputed structural score of a route candidate.
// Comparison between two scores uses the following precedence (highest to lowest):
//  1. LiteralSegments  — more is better
//  2. ConstrainedVars  — more is better
//  3. BroadVars        — fewer is better (negate for comparison)
//  4. QueryMatches     — more is better
//  5. Priority         — higher is better
//  6. Registration     — lower (earlier) index is better
type candidateScore struct {
	LiteralSegments int
	ConstrainedVars int
	BroadVars       int
	QueryMatches    int
	Priority        int
	Registration    int
}

func (s candidateScore) beats(other candidateScore) bool {
	if s.LiteralSegments != other.LiteralSegments {
		return s.LiteralSegments > other.LiteralSegments
	}
	if s.ConstrainedVars != other.ConstrainedVars {
		return s.ConstrainedVars > other.ConstrainedVars
	}
	if s.BroadVars != other.BroadVars {
		return s.BroadVars < other.BroadVars
	}
	if s.QueryMatches != other.QueryMatches {
		return s.QueryMatches > other.QueryMatches
	}
	if s.Priority != other.Priority {
		return s.Priority > other.Priority
	}
	return s.Registration < other.Registration
}

// Match is the result of a successful route resolution. It is stored in the
// request context and accessible via [MatchFromContext], [RouteNameFromContext],
// and [ParamsFromContext].
type Match struct {
	// Route is the selected [Route] definition.
	Route *Route

	// Name is a copy of Route.Name provided for convenient access.
	Name string

	// Params holds all resolved route parameters: values extracted from the
	// request URL plus defaults for any absent variables.
	Params Params

	// Method is the normalized HTTP method string of the matched request.
	Method string

	// CanonicalURL is the URL produced by re-expanding the matched route
	// template with the resolved Params. It is nil when CanonicalPolicy is
	// CanonicalIgnore.
	CanonicalURL *url.URL

	// IsCanonical reports whether the request URL equals the canonical URL.
	// It is meaningful only when CanonicalURL is non-nil.
	IsCanonical bool

	// RedirectNeeded reports whether the router should issue a redirect to
	// CanonicalURL instead of invoking the route handler.
	RedirectNeeded bool

	score candidateScore
}
