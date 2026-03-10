package dispatch

import "net/http"

// WithNotFoundHandler sets a custom handler called when no route matches.
// Defaults to http.NotFoundHandler().
func WithNotFoundHandler(h http.Handler) Option {
	return func(cfg *routerConfig) { cfg.notFoundHandler = h }
}

// WithMethodNotAllowedHandler sets a custom handler called when a URL matches
// structurally but no registered route allows the request method.
// Defaults to a handler that writes "405 Method Not Allowed".
func WithMethodNotAllowedHandler(h http.Handler) Option {
	return func(cfg *routerConfig) { cfg.methodNotAllowedHandler = h }
}

// WithErrorHandler sets the handler invoked on internal dispatch errors.
func WithErrorHandler(h http.Handler) Option {
	return func(cfg *routerConfig) { cfg.dispatchErrorHandler = h }
}

// WithDefaultQueryMode sets the QueryMode applied to routes whose own
// QueryMode is zero. Defaults to QueryLoose.
func WithDefaultQueryMode(m QueryMode) Option {
	return func(cfg *routerConfig) { cfg.defaultQueryMode = m }
}

// WithDefaultCanonicalPolicy sets the CanonicalPolicy applied to routes whose
// own CanonicalPolicy is zero. Defaults to CanonicalIgnore.
func WithDefaultCanonicalPolicy(p CanonicalPolicy) Option {
	return func(cfg *routerConfig) { cfg.defaultCanonicalPolicy = p }
}

// WithDefaultRedirectCode sets the HTTP redirect status code used for
// canonical redirects when a route does not specify its own RedirectCode.
// Must be a 3xx status code. Defaults to http.StatusMovedPermanently (301).
func WithDefaultRedirectCode(code int) Option {
	return func(cfg *routerConfig) { cfg.defaultRedirectCode = code }
}

// WithImplicitHEAD controls whether GET routes also match HEAD requests.
// Enabled by default.
func WithImplicitHEAD(enabled bool) Option {
	return func(cfg *routerConfig) { cfg.implicitHEADFromGET = enabled }
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
