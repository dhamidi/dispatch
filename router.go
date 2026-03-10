package dispatch

import "net/http"

// routerConfig holds all configurable router settings.
type routerConfig struct {
	notFoundHandler         http.Handler
	methodNotAllowedHandler http.Handler
	dispatchErrorHandler    http.Handler
	defaultQueryMode        QueryMode
	defaultCanonicalPolicy  CanonicalPolicy
	defaultSlashPolicy      SlashPolicy
	defaultRedirectCode     int
	implicitHEADFromGET     bool
}

// Router is a semantic HTTP router. It implements [http.Handler].
//
// Build a Router with [New], register routes with [Router.Handle] or the
// convenience methods, then pass the Router to any net/http server.
//
// A Router is not safe for concurrent registration after serving has begun.
// Register all routes during startup before calling [http.ListenAndServe].
type Router struct {
	config routerConfig
	routes []*registeredRoute            // in registration order
	byName map[string]*registeredRoute
}

// registeredRoute wraps a Route with precomputed scoring metadata.
type registeredRoute struct {
	Route
	index int            // registration order index, used in candidateScore.Registration
	score candidateScore // precomputed structural scoring hints
}

// Option is a functional option for configuring a [Router] at construction time.
type Option func(*routerConfig)

// New creates a new Router with optional configuration options.
//
// Example:
//
//	r := dispatch.New(
//	    dispatch.WithNotFoundHandler(myNotFound),
//	    dispatch.WithDefaultQueryMode(dispatch.QueryStrict),
//	)
func New(opts ...Option) *Router {
	cfg := routerConfig{
		defaultRedirectCode: http.StatusMovedPermanently,
		implicitHEADFromGET: true,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.notFoundHandler == nil {
		cfg.notFoundHandler = http.HandlerFunc(defaultNotFound)
	}
	if cfg.methodNotAllowedHandler == nil {
		cfg.methodNotAllowedHandler = http.HandlerFunc(defaultMethodNotAllowed)
	}
	return &Router{
		config: cfg,
		byName: make(map[string]*registeredRoute),
	}
}

// Scope registers routes under a shared configuration scope (§9).
// Optional ScopeOption values configure the scope before fn is called.
func (r *Router) Scope(fn func(*Scope), opts ...ScopeOption) {
	s := &Scope{router: r}
	for _, o := range opts {
		o(s)
	}
	fn(s)
}

// WithScope creates a detached Scope with the provided options that can be
// used to register routes individually.
func (r *Router) WithScope(opts ...ScopeOption) *Scope {
	s := &Scope{router: r}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// --- internal helpers -------------------------------------------------------

// defaultNotFound writes a plain 404 response.
func defaultNotFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 not found", http.StatusNotFound)
}

// defaultMethodNotAllowed writes a plain 405 response.
func defaultMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}
