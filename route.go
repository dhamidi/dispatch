package dispatch

import (
	"net/http"

	"github.com/dhamidi/uritemplate"
)

// Route defines a semantic HTTP endpoint. A Route is the single source of
// truth for both inbound request matching and outbound URL generation.
//
// Register routes via [Router.Handle], [Router.MustHandle], or the
// convenience methods [Router.GET], [Router.POST], etc.
type Route struct {
	// Name is the application-defined identifier for this route. It must be
	// non-empty and unique within a [Router]. Names should be stable across
	// process restarts because they are used for URL generation.
	//
	// Examples: "users.show", "search", "admin.reports.download"
	Name string

	// Methods is the set of HTTP methods this route accepts. It must not be
	// zero. Use the MethodSet constants (GET, POST, PUT, etc.) combined with
	// bitwise OR.
	Methods MethodSet

	// Template is the parsed URI template used for both reverse matching and
	// URL generation. It must be non-nil and must originate from the
	// github.com/dhamidi/uritemplate package.
	Template *uritemplate.Template

	// Handler is the http.Handler invoked when this route is selected. It
	// must be non-nil for routes that participate in dispatch.
	Handler http.Handler

	// Defaults provides fallback values for template variables that are
	// absent in the matched URL. Defaults are applied after extraction and
	// before constraints; they never override extracted values.
	Defaults Params

	// Constraints is a list of post-extraction validation rules. Constraints
	// are evaluated in slice order after defaults are applied; the first
	// failure rejects the candidate.
	Constraints []Constraint

	// QueryMode controls how undeclared query parameters affect matching.
	// Zero value is equivalent to QueryLoose unless the router has a default.
	QueryMode QueryMode

	// CanonicalPolicy controls behaviour when the inbound URL differs from
	// the canonical URL computed from this route. Zero value is equivalent to
	// CanonicalIgnore unless the router has a default.
	CanonicalPolicy CanonicalPolicy

	// RedirectCode is the HTTP status code used for canonical redirects.
	// When zero and CanonicalPolicy is CanonicalRedirect, the router uses its
	// configured default (typically http.StatusMovedPermanently).
	RedirectCode int

	// Priority is an explicit tie-breaker used during candidate scoring.
	// Higher values win. When equal, earlier registration order wins.
	Priority int

	// Metadata is an opaque map of application-defined string key-value
	// pairs. The router treats it as read-only after registration.
	Metadata map[string]string
}
