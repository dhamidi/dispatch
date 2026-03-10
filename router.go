package dispatch

import "net/http"

// routerConfig holds all configurable router settings.
type routerConfig struct {
	notFoundHandler         http.Handler
	methodNotAllowedHandler http.Handler
	dispatchErrorHandler    http.Handler
	defaultQueryMode        QueryMode
	defaultCanonicalPolicy  CanonicalPolicy
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

// Route returns the registered Route for the given name.
func (r *Router) Route(name string) (*Route, bool) {
	reg, ok := r.byName[name]
	if !ok {
		return nil, false
	}
	return &reg.Route, true
}

// Routes returns read-only summaries of all registered routes (§15).
func (r *Router) Routes() []RouteInfo {
	infos := make([]RouteInfo, len(r.routes))
	for i, reg := range r.routes {
		var meta map[string]string
		if reg.Metadata != nil {
			meta = make(map[string]string, len(reg.Metadata))
			for k, v := range reg.Metadata {
				meta[k] = v
			}
		}
		infos[i] = RouteInfo{
			Name:     reg.Name,
			Template: reg.Template.String(),
			Methods:  reg.Methods,
			Metadata: meta,
		}
	}
	return infos
}

// Scope registers routes under a shared configuration scope (§9).
func (r *Router) Scope(fn func(*Scope)) {
	s := &Scope{router: r}
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

// RouteInfo is a read-only summary of a registered route for introspection
// and debugging (§15).
type RouteInfo struct {
	Name     string
	Template string
	Methods  MethodSet
	Metadata map[string]string
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
