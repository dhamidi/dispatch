package dispatch

import "fmt"

// QueryMode controls how query string variables in a route template
// participate in request matching.
type QueryMode uint8

const (
	// QueryLoose matches when declared query template variables satisfy the
	// request; undeclared query parameters are ignored and do not prevent a
	// match.
	QueryLoose QueryMode = iota

	// QueryCanonical matches like QueryLoose but additionally normalises the
	// set of query parameters when computing the canonical URL; undeclared
	// parameters may be dropped from the canonical form.
	QueryCanonical

	// QueryStrict rejects a candidate route when the request contains query
	// parameters that are not declared in the route template.
	QueryStrict
)

// String returns the name of the QueryMode constant or a numeric
// representation for unknown values.
func (m QueryMode) String() string {
	switch m {
	case QueryLoose:
		return "QueryLoose"
	case QueryCanonical:
		return "QueryCanonical"
	case QueryStrict:
		return "QueryStrict"
	default:
		return fmt.Sprintf("QueryMode(%d)", m)
	}
}

// CanonicalPolicy controls the router's behaviour when a matched request URL
// differs from the canonical URL computed from the matched route.
type CanonicalPolicy uint8

const (
	// CanonicalIgnore disables canonical URL computation entirely. The router
	// dispatches to the handler without comparing canonical form.
	CanonicalIgnore CanonicalPolicy = iota

	// CanonicalAnnotate computes the canonical URL and records it in the
	// [Match] returned from context helpers, but does not redirect or reject
	// non-canonical requests.
	CanonicalAnnotate

	// CanonicalRedirect issues an HTTP redirect to the canonical URL when the
	// request is non-canonical. The matched handler is not invoked.
	CanonicalRedirect

	// CanonicalReject treats non-canonical requests as unmatched, preventing
	// dispatch to the handler.
	CanonicalReject
)

// SlashPolicy controls trailing-slash normalization behavior.
type SlashPolicy uint8

const (
	// SlashIgnore performs no trailing-slash normalization. A request for
	// /admin will not match a route registered as /admin/ and vice versa.
	SlashIgnore SlashPolicy = iota

	// SlashRedirect enables trailing-slash normalization. When a request
	// does not match any route but would match with the trailing slash
	// toggled, the router issues an HTTP redirect to the corrected URL.
	// For parameterized routes, a trailing slash in the request path is
	// normalized before matching to prevent it from being absorbed into a
	// path parameter.
	SlashRedirect
)

// String returns the name of the SlashPolicy constant or a numeric
// representation for unknown values.
func (p SlashPolicy) String() string {
	switch p {
	case SlashIgnore:
		return "SlashIgnore"
	case SlashRedirect:
		return "SlashRedirect"
	default:
		return fmt.Sprintf("SlashPolicy(%d)", p)
	}
}

// String returns the name of the CanonicalPolicy constant or a numeric
// representation for unknown values.
func (p CanonicalPolicy) String() string {
	switch p {
	case CanonicalIgnore:
		return "CanonicalIgnore"
	case CanonicalAnnotate:
		return "CanonicalAnnotate"
	case CanonicalRedirect:
		return "CanonicalRedirect"
	case CanonicalReject:
		return "CanonicalReject"
	default:
		return fmt.Sprintf("CanonicalPolicy(%d)", p)
	}
}
