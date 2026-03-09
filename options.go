package dispatch

import "net/http"

// Option configures a Router at construction time.
type Option func(*Router)

// WithNotFoundHandler sets the handler invoked when no route matches.
func WithNotFoundHandler(h http.Handler) Option {
	return func(r *Router) { r.notFound = h }
}

// WithMethodNotAllowedHandler sets the handler invoked when a URL matches but
// the HTTP method is not allowed.
func WithMethodNotAllowedHandler(h http.Handler) Option {
	return func(r *Router) { r.methodNotAllowed = h }
}

// WithErrorHandler sets the handler invoked on internal dispatch errors.
func WithErrorHandler(h http.Handler) Option {
	return func(r *Router) { r.dispatchError = h }
}

// WithDefaultQueryMode sets the router-level default QueryMode used when a
// route does not specify its own.
func WithDefaultQueryMode(qm QueryMode) Option {
	return func(r *Router) { r.defaultQueryMode = qm }
}

// WithDefaultCanonicalPolicy sets the router-level default CanonicalPolicy.
func WithDefaultCanonicalPolicy(cp CanonicalPolicy) Option {
	return func(r *Router) { r.defaultCanonicalPolicy = cp }
}

// WithDefaultRedirectCode sets the router-level default HTTP redirect status
// code used for canonical redirects when a route does not specify one.
func WithDefaultRedirectCode(code int) Option {
	return func(r *Router) { r.defaultRedirectCode = code }
}

// WithImplicitHEAD controls whether GET routes also match HEAD requests.
// Enabled by default.
func WithImplicitHEAD(enabled bool) Option {
	return func(r *Router) { r.implicitHEADFromGET = enabled }
}

// RouteOption configures an individual Route during convenience registration.
type RouteOption func(*Route)

// WithDefaults sets default parameter values for the route.
// Defaults are applied after extraction and never override extracted values.
func WithDefaults(params Params) RouteOption {
	return func(route *Route) { route.Defaults = params.Clone() }
}

// WithConstraint appends a single Constraint to the route.
func WithConstraint(c Constraint) RouteOption {
	return func(route *Route) { route.Constraints = append(route.Constraints, c) }
}

// WithConstraints appends multiple Constraints to the route.
func WithConstraints(cs ...Constraint) RouteOption {
	return func(route *Route) { route.Constraints = append(route.Constraints, cs...) }
}

// WithQueryMode sets the QueryMode for the route.
func WithQueryMode(qm QueryMode) RouteOption {
	return func(route *Route) { route.QueryMode = qm }
}

// WithCanonicalPolicy sets the CanonicalPolicy for the route.
func WithCanonicalPolicy(cp CanonicalPolicy) RouteOption {
	return func(route *Route) { route.CanonicalPolicy = cp }
}

// WithRedirectCode sets the HTTP status code used for canonical redirects.
func WithRedirectCode(code int) RouteOption {
	return func(route *Route) { route.RedirectCode = code }
}

// WithPriority sets the explicit priority tie-breaker for the route.
// Higher values win over lower values after structural scoring.
func WithPriority(p int) RouteOption {
	return func(route *Route) { route.Priority = p }
}

// WithMetadata sets a single metadata key-value pair on the route.
func WithMetadata(key, value string) RouteOption {
	return func(route *Route) {
		if route.Metadata == nil {
			route.Metadata = make(map[string]string)
		}
		route.Metadata[key] = value
	}
}
