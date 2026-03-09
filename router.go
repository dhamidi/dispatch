package dispatch

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/dhamidi/uritemplate"
)

// Router is the central HTTP router.
// It implements http.Handler and is safe for concurrent use after all routes
// have been registered.
type Router struct {
	// routes stores routes in registration order for candidate scanning.
	routes []*Route

	// routesByName maps route names to routes for O(1) generation lookups.
	routesByName map[string]*Route

	// Error handlers (configurable via Options).
	notFound         http.Handler
	methodNotAllowed http.Handler
	dispatchError    http.Handler

	// Router-level defaults (may be overridden per route).
	defaultQueryMode       QueryMode
	defaultCanonicalPolicy CanonicalPolicy
	defaultRedirectCode    int

	// implicitHEADFromGET: when true, GET routes also match HEAD requests.
	implicitHEADFromGET bool
}

// New creates a new Router configured with the provided Options.
func New(opts ...Option) *Router {
	r := &Router{
		routesByName:        make(map[string]*Route),
		implicitHEADFromGET: true,
		defaultRedirectCode: http.StatusMovedPermanently,
		notFound:            http.HandlerFunc(defaultNotFound),
		methodNotAllowed:    http.HandlerFunc(defaultMethodNotAllowed),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m, err := r.Match(req)
	if err != nil {
		switch err {
		case ErrNotFound:
			r.notFound.ServeHTTP(w, req)
		case ErrMethodNotAllowed:
			r.methodNotAllowed.ServeHTTP(w, req)
		default:
			if r.dispatchError != nil {
				r.dispatchError.ServeHTTP(w, req)
			} else {
				http.Error(w, "500 internal server error", http.StatusInternalServerError)
			}
		}
		return
	}

	if m.RedirectNeeded && m.CanonicalURL != nil {
		code := m.Route.RedirectCode
		if code == 0 {
			code = r.defaultRedirectCode
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

// Handle registers a route.
func (r *Router) Handle(route Route) error {
	if route.Name == "" {
		return ErrEmptyRouteName
	}
	if _, exists := r.routesByName[route.Name]; exists {
		return ErrDuplicateRoute
	}
	if route.Template == nil {
		return ErrNilTemplate
	}
	if route.Handler == nil {
		return ErrNilHandler
	}
	if route.Methods == 0 {
		return ErrInvalidMethodSet
	}
	if route.CanonicalPolicy == CanonicalRedirect && route.RedirectCode != 0 {
		if route.RedirectCode < 300 || route.RedirectCode > 399 {
			return ErrInvalidRedirectCode
		}
	}

	// Clone mutable fields to avoid aliasing
	stored := route
	stored.Defaults = route.Defaults.Clone()
	if route.Metadata != nil {
		stored.Metadata = make(map[string]string, len(route.Metadata))
		for k, v := range route.Metadata {
			stored.Metadata[k] = v
		}
	}
	if route.Constraints != nil {
		stored.Constraints = make([]Constraint, len(route.Constraints))
		copy(stored.Constraints, route.Constraints)
	}

	r.routes = append(r.routes, &stored)
	r.routesByName[stored.Name] = &stored
	return nil
}

// MustHandle calls Handle and panics if it returns an error.
func (r *Router) MustHandle(route Route) {
	if err := r.Handle(route); err != nil {
		panic(err)
	}
}

// Match resolves req to the best matching route.
func (r *Router) Match(req *http.Request) (*Match, error) {
	rc := &RequestContext{
		Request: req,
		URL:     req.URL,
		Method:  req.Method,
		Host:    req.Host,
	}

	candidates, methodMismatch := r.filterCandidates(rc)
	if len(candidates) == 0 {
		if methodMismatch {
			return nil, ErrMethodNotAllowed
		}
		return nil, ErrNotFound
	}

	best := r.selectBest(candidates)
	if best == nil {
		return nil, ErrNotFound
	}

	m := &Match{
		Route:  best.route,
		Name:   best.route.Name,
		Params: best.params,
		Method: rc.Method,
		score:  best.score,
	}

	// Compute canonical URL if applicable
	policy := best.route.CanonicalPolicy
	if policy == CanonicalIgnore {
		policy = r.defaultCanonicalPolicy
	}
	if policy != CanonicalIgnore {
		canonical, err := r.computeCanonical(best.route, best.params)
		if err == nil && canonical != nil {
			m.CanonicalURL = canonical
			m.IsCanonical = isCanonicalMatch(rc.URL, canonical)
			if !m.IsCanonical {
				switch policy {
				case CanonicalRedirect:
					m.RedirectNeeded = true
				case CanonicalReject:
					return nil, ErrNotFound
				}
			}
		}
	}

	return m, nil
}

// URL generates the full URL for the named route expanded with params (§12).
func (r *Router) URL(name string, params Params) (*url.URL, error) {
	route, ok := r.routesByName[name]
	if !ok {
		return nil, ErrUnknownRoute
	}

	merged := mergeParams(params, route.Defaults)
	vals := paramsToValues(merged)

	expanded, err := route.Template.Expand(vals)
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
	route, ok := r.routesByName[name]
	return route, ok
}

// Routes returns read-only summaries of all registered routes (§15).
func (r *Router) Routes() []RouteInfo {
	infos := make([]RouteInfo, len(r.routes))
	for i, rt := range r.routes {
		var meta map[string]string
		if rt.Metadata != nil {
			meta = make(map[string]string, len(rt.Metadata))
			for k, v := range rt.Metadata {
				meta[k] = v
			}
		}
		infos[i] = RouteInfo{
			Name:     rt.Name,
			Template: rt.Template.String(),
			Methods:  rt.Methods,
			Metadata: meta,
		}
	}
	return infos
}

// GET registers a route that matches the GET (and implicitly HEAD) method.
func (r *Router) GET(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodGET, name, tmpl, h, opts...)
}

// POST registers a route that matches the POST method.
func (r *Router) POST(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodPOST, name, tmpl, h, opts...)
}

// PUT registers a route that matches the PUT method.
func (r *Router) PUT(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodPUT, name, tmpl, h, opts...)
}

// PATCH registers a route that matches the PATCH method.
func (r *Router) PATCH(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodPATCH, name, tmpl, h, opts...)
}

// DELETE registers a route that matches the DELETE method.
func (r *Router) DELETE(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodDELETE, name, tmpl, h, opts...)
}

// OPTIONS registers a route that matches the OPTIONS method.
func (r *Router) OPTIONS(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.register(MethodOPTIONS, name, tmpl, h, opts...)
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

// candidate is the internal representation of a route during match resolution.
type candidate struct {
	route  *Route
	params Params
	score  candidateScore
}

// filterCandidates returns all routes that pass method + template matching.
// It also detects method-not-allowed situations.
func (r *Router) filterCandidates(rc *RequestContext) (matched []*candidate, methodMismatch bool) {
	reqMethod := methodFromString(rc.Method)

	// Build the request URI for template matching
	matchURI := rc.URL.RequestURI()

	for i, route := range r.routes {
		// Attempt URI template reverse match
		vals, ok := route.Template.Match(matchURI)
		if !ok {
			// Also try matching with just the path (without query) for templates that don't have query expressions
			vals, ok = route.Template.FromURL(rc.URL)
			if !ok {
				continue
			}
		}

		// Template matched structurally - check method compatibility
		methodOK := route.Methods.contains(reqMethod)
		if !methodOK && r.implicitHEADFromGET && reqMethod == MethodHEAD {
			methodOK = route.Methods.contains(MethodGET)
		}
		if !methodOK {
			methodMismatch = true
			continue
		}

		// Convert Values to Params
		params := valuesToParams(vals)

		// Apply defaults (never override extracted values)
		if route.Defaults != nil {
			for k, v := range route.Defaults {
				if _, exists := params[k]; !exists {
					params[k] = v
				}
			}
		}

		// Enforce QueryMode
		qm := route.QueryMode
		if qm == QueryLoose {
			qm = r.defaultQueryMode
		}
		if qm == QueryStrict {
			// Check for undeclared query parameters
			declaredVars := templateVarNames(route.Template)
			for key := range rc.URL.Query() {
				if _, declared := declaredVars[key]; !declared {
					goto nextRoute
				}
			}
		}

		// Evaluate constraints
		{
			constraintOK := true
			for _, c := range route.Constraints {
				if !c.Check(rc, params) {
					constraintOK = false
					break
				}
			}
			if !constraintOK {
				continue
			}
		}

		// Compute score
		{
			score := computeScore(route, params, rc.URL, i)
			matched = append(matched, &candidate{
				route:  route,
				params: params,
				score:  score,
			})
		}
		continue

	nextRoute:
	}
	return matched, methodMismatch
}

// selectBest picks the highest-scoring candidate deterministically (§10.7).
func (r *Router) selectBest(candidates []*candidate) *candidate {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if best.score.less(c.score) {
			best = c
		}
	}
	return best
}

// computeCanonical expands the matched route template with final params and
// returns the canonical *url.URL (§11.1).
func (r *Router) computeCanonical(route *Route, params Params) (*url.URL, error) {
	vals := paramsToValues(params)
	expanded, err := route.Template.Expand(vals)
	if err != nil {
		return nil, err
	}
	return url.Parse(expanded)
}

// defaultNotFound writes a plain 404 response.
func defaultNotFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 not found", http.StatusNotFound)
}

// defaultMethodNotAllowed writes a plain 405 response.
func defaultMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
}

// --- utility functions ------------------------------------------------------

// paramsToValues converts Params to uritemplate.Values.
func paramsToValues(p Params) uritemplate.Values {
	vals := make(uritemplate.Values, len(p))
	for k, v := range p {
		vals[k] = uritemplate.String(v)
	}
	return vals
}

// valuesToParams converts uritemplate.Values to Params by expanding each value
// through a single-variable template.
func valuesToParams(vals uritemplate.Values) Params {
	if vals == nil {
		return make(Params)
	}
	p := make(Params, len(vals))
	for k, v := range vals {
		// Use a simple template to extract the string value
		t := uritemplate.MustParse("{v}")
		expanded, err := t.Expand(uritemplate.Values{"v": v})
		if err == nil {
			p[k] = expanded
		}
	}
	return p
}

// mergeParams merges provided params with defaults. Provided params take precedence.
func mergeParams(provided, defaults Params) Params {
	result := make(Params)
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range provided {
		result[k] = v
	}
	return result
}

// templateVarNames extracts variable names declared in a template by parsing
// the raw template string.
func templateVarNames(t *uritemplate.Template) map[string]struct{} {
	raw := t.String()
	vars := make(map[string]struct{})
	i := 0
	for i < len(raw) {
		if raw[i] == '{' {
			end := strings.IndexByte(raw[i:], '}')
			if end < 0 {
				break
			}
			body := raw[i+1 : i+end]
			// Skip operator character if present
			if len(body) > 0 {
				first := body[0]
				if first == '+' || first == '#' || first == '.' || first == '/' || first == ';' || first == '?' || first == '&' {
					body = body[1:]
				}
			}
			// Split by comma for multiple variables
			parts := strings.Split(body, ",")
			for _, p := range parts {
				// Remove modifiers (:N, *)
				name := strings.TrimRight(p, "*")
				if colonIdx := strings.IndexByte(name, ':'); colonIdx >= 0 {
					name = name[:colonIdx]
				}
				name = strings.TrimSpace(name)
				if name != "" {
					vars[name] = struct{}{}
				}
			}
			i += end + 1
		} else {
			i++
		}
	}
	return vars
}

// computeScore computes the candidateScore for a route given its match.
func computeScore(route *Route, params Params, reqURL *url.URL, registrationIdx int) candidateScore {
	raw := route.Template.String()

	// Count literal segments (non-expression parts split by /)
	literalCount := 0
	inExpr := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '{' {
			inExpr = true
		} else if raw[i] == '}' {
			inExpr = false
		} else if !inExpr && raw[i] == '/' {
			literalCount++
		} else if !inExpr && raw[i] != '/' && raw[i] != '?' {
			// Count non-separator literal characters
		}
	}
	// Also count literal characters for more precision
	litChars := 0
	inExpr = false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '{' {
			inExpr = true
		} else if raw[i] == '}' {
			inExpr = false
		} else if !inExpr {
			litChars++
		}
	}

	// Count query matches
	queryMatches := 0
	vars := templateVarNames(route.Template)
	if reqURL != nil {
		for key := range reqURL.Query() {
			if _, ok := vars[key]; ok {
				queryMatches++
			}
		}
	}

	// Constrained vars = number of constraints
	constrainedVars := len(route.Constraints)

	// Broad vars = total template vars minus constrained
	broadVars := len(vars) - constrainedVars
	if broadVars < 0 {
		broadVars = 0
	}

	return candidateScore{
		LiteralSegments: litChars,
		ConstrainedVars: constrainedVars,
		BroadVars:       broadVars,
		QueryMatches:    queryMatches,
		Priority:        route.Priority,
		Registration:    registrationIdx,
	}
}

// isCanonicalMatch compares the request URL with the canonical URL.
func isCanonicalMatch(reqURL, canonical *url.URL) bool {
	if reqURL == nil || canonical == nil {
		return true
	}
	// Compare path and query
	reqPath := reqURL.Path
	canonPath := canonical.Path
	if reqPath != canonPath {
		return false
	}
	// Compare query parameters (order-independent)
	reqQuery := reqURL.Query()
	canonQuery := canonical.Query()
	if len(reqQuery) != len(canonQuery) {
		return false
	}
	for k, rv := range reqQuery {
		cv, ok := canonQuery[k]
		if !ok || len(rv) != len(cv) {
			return false
		}
		for i := range rv {
			if rv[i] != cv[i] {
				return false
			}
		}
	}
	return true
}
