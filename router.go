package dispatch

import (
	"net/http"
	"net/url"

	"github.com/dhamidi/uritemplate"
)

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

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m, err := r.Match(req)
	if err != nil {
		switch err {
		case ErrNotFound:
			r.config.notFoundHandler.ServeHTTP(w, req)
		case ErrMethodNotAllowed:
			r.config.methodNotAllowedHandler.ServeHTTP(w, req)
		default:
			if r.config.dispatchErrorHandler != nil {
				r.config.dispatchErrorHandler.ServeHTTP(w, req)
			} else {
				http.Error(w, "500 internal server error", http.StatusInternalServerError)
			}
		}
		return
	}

	if m.RedirectNeeded && m.CanonicalURL != nil {
		code := m.Route.RedirectCode
		if code == 0 {
			code = r.config.defaultRedirectCode
		}
		if code == 0 {
			code = http.StatusMovedPermanently
		}
		http.Redirect(w, req, m.CanonicalURL.String(), code)
		return
	}

	ctx := withMatch(req.Context(), m)
	m.Route.Handler.ServeHTTP(w, req.WithContext(ctx))
}

// URL generates the full URL for the named route expanded with params (§12).
func (r *Router) URL(name string, params Params) (*url.URL, error) {
	reg, ok := r.byName[name]
	if !ok {
		return nil, ErrUnknownRoute
	}

	merged := mergeParams(params, reg.Defaults)
	vals := paramsToValues(merged)

	expanded, err := reg.Template.Expand(vals)
	if err != nil {
		return nil, ErrMissingParam
	}

	u, err := url.Parse(expanded)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Path generates the path string for the named route expanded with params.
func (r *Router) Path(name string, params Params) (string, error) {
	u, err := r.URL(name, params)
	if err != nil {
		return "", err
	}
	result := u.Path
	if u.RawQuery != "" {
		result += "?" + u.RawQuery
	}
	return result, nil
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

// GET registers a route that matches the GET (and implicitly HEAD) method.
func (r *Router) GET(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(GET, name, tmpl, h, opts...)
}

// POST registers a route that matches the POST method.
func (r *Router) POST(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(POST, name, tmpl, h, opts...)
}

// PUT registers a route that matches the PUT method.
func (r *Router) PUT(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(PUT, name, tmpl, h, opts...)
}

// PATCH registers a route that matches the PATCH method.
func (r *Router) PATCH(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(PATCH, name, tmpl, h, opts...)
}

// DELETE registers a route that matches the DELETE method.
func (r *Router) DELETE(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(DELETE, name, tmpl, h, opts...)
}

// OPTIONS registers a route that matches the OPTIONS method.
func (r *Router) OPTIONS(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(OPTIONS, name, tmpl, h, opts...)
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

// register is the shared implementation for GET/POST/PUT/PATCH/DELETE/OPTIONS.
func (r *Router) register(methods MethodSet, name, tmpl string, h http.Handler, opts ...RouteOption) error {
	t, err := uritemplate.Parse(tmpl)
	if err != nil {
		return err
	}
	route := Route{Name: name, Methods: methods, Template: t, Handler: h}
	for _, opt := range opts {
		opt(&route)
	}
	return r.Handle(route)
}

// defaultNotFound writes a plain 404 response.
func defaultNotFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 not found", http.StatusNotFound)
}

// defaultMethodNotAllowed writes a plain 405 response.
func defaultMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}
