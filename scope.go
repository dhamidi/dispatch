package dispatch

import (
	"net/http"
	"strings"

	"github.com/dhamidi/uritemplate"
)

// Scope supports grouped route registration with shared name prefixes,
// template prefixes, defaults, constraints, query modes, canonical policies,
// and metadata (§9).
//
// Scopes may be nested; inner values override outer values for defaults and
// metadata while constraints are appended outer-first (§9.3.1).
type Scope struct {
	router *Router
	parent *Scope // nil for top-level scopes

	// namePrefix is prepended to route names (separator: ".").
	namePrefix string

	// templatePrefix is prepended to route templates before parsing.
	templatePrefix string

	// defaults provides inherited default params for all routes in this scope.
	defaults Params

	// constraints are inherited by all routes in this scope (outer-first).
	constraints []Constraint

	// queryMode is the inherited query mode (0 means use router default).
	queryMode QueryMode

	// canonicalPolicy is the inherited canonical policy (0 means use router default).
	canonicalPolicy CanonicalPolicy

	// metadata is inherited by all routes in this scope (inner overrides outer).
	metadata map[string]string
}

// ScopeOption configures a Scope.
type ScopeOption func(*Scope)

// WithNamePrefix sets the name prefix for the scope.
// Names are composed with "." as separator (e.g. "admin" + "users.show" -> "admin.users.show").
// A trailing "." in prefix is stripped to avoid double separators.
func WithNamePrefix(prefix string) ScopeOption {
	return func(s *Scope) { s.namePrefix = strings.TrimRight(prefix, ".") }
}

// WithTemplatePrefix sets the URI template prefix prepended to route templates.
func WithTemplatePrefix(prefix string) ScopeOption {
	return func(s *Scope) { s.templatePrefix = prefix }
}

// WithScopeDefaults merges params into the scope's inherited defaults.
// Inner values override outer values.
func WithScopeDefaults(params Params) ScopeOption {
	return func(s *Scope) {
		if s.defaults == nil {
			s.defaults = make(Params)
		}
		for k, v := range params {
			s.defaults[k] = v
		}
	}
}

// WithScopeConstraint appends c to the scope's inherited constraints.
// Constraints are evaluated outer-first.
func WithScopeConstraint(c Constraint) ScopeOption {
	return func(s *Scope) { s.constraints = append(s.constraints, c) }
}

// WithScopeQueryMode sets the inherited QueryMode for routes in this scope.
func WithScopeQueryMode(qm QueryMode) ScopeOption {
	return func(s *Scope) { s.queryMode = qm }
}

// WithScopeCanonicalPolicy sets the inherited CanonicalPolicy.
func WithScopeCanonicalPolicy(cp CanonicalPolicy) ScopeOption {
	return func(s *Scope) { s.canonicalPolicy = cp }
}

// WithScopeMetadata adds a metadata key-value pair inherited by routes in
// this scope. Inner values override outer values at route registration time.
func WithScopeMetadata(key, value string) ScopeOption {
	return func(s *Scope) {
		if s.metadata == nil {
			s.metadata = make(map[string]string)
		}
		s.metadata[key] = value
	}
}

// Handle registers a route within the scope, applying scope-level
// configuration before delegating to Router.Handle.
func (s *Scope) Handle(route Route) error {
	s.applyToRoute(&route)
	return s.router.Handle(route)
}

// GET registers a GET route within the scope.
func (s *Scope) GET(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return s.register(GET, name, tmpl, h, opts...)
}

// POST registers a POST route within the scope.
func (s *Scope) POST(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return s.register(POST, name, tmpl, h, opts...)
}

// PUT registers a PUT route within the scope.
func (s *Scope) PUT(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return s.register(PUT, name, tmpl, h, opts...)
}

// PATCH registers a PATCH route within the scope.
func (s *Scope) PATCH(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return s.register(PATCH, name, tmpl, h, opts...)
}

// DELETE registers a DELETE route within the scope.
func (s *Scope) DELETE(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return s.register(DELETE, name, tmpl, h, opts...)
}

// Scope creates a nested child scope that inherits from s and is further
// configured by fn. Nested name/template prefixes are concatenated;
// defaults and metadata are merged (inner wins); constraints are appended
// outer-first (§9.3.1).
func (s *Scope) Scope(fn func(*Scope), opts ...ScopeOption) {
	child := &Scope{
		router:          s.router,
		parent:          s,
		namePrefix:      "",
		templatePrefix:  "",
		queryMode:       s.queryMode,
		canonicalPolicy: s.canonicalPolicy,
	}
	if s.defaults != nil {
		child.defaults = s.defaults.Clone()
	}
	if s.constraints != nil {
		child.constraints = make([]Constraint, len(s.constraints))
		copy(child.constraints, s.constraints)
	}
	if s.metadata != nil {
		child.metadata = make(map[string]string, len(s.metadata))
		for k, v := range s.metadata {
			child.metadata[k] = v
		}
	}
	for _, o := range opts {
		o(child)
	}
	// Compose name prefix: parent.child
	if s.namePrefix != "" && child.namePrefix != "" {
		child.namePrefix = s.namePrefix + "." + child.namePrefix
	} else if s.namePrefix != "" {
		child.namePrefix = s.namePrefix
	}
	// Compose template prefix: parent + child (string concatenation)
	child.templatePrefix = s.templatePrefix + child.templatePrefix
	fn(child)
}

// --- internal helpers -------------------------------------------------------

// register is the shared implementation for the convenience method helpers.
func (s *Scope) register(methods MethodSet, name, tmpl string, h http.Handler, opts ...RouteOption) error {
	fullName := s.effectiveName(name)
	fullTmpl := s.effectiveTemplate(tmpl)
	t, err := uritemplate.Parse(fullTmpl)
	if err != nil {
		return err
	}
	route := Route{Name: fullName, Methods: methods, Template: t, Handler: h}
	for _, opt := range opts {
		opt(&route)
	}
	s.applyToRoute(&route)
	return s.router.Handle(route)
}

// applyToRoute merges scope-level configuration into route before registration.
//
// Merge rules (§9.3.1):
//   - defaults: scope provides fallback; route-level opts override scope.
//   - constraints: scope constraints are prepended (outer-first).
//   - queryMode / canonicalPolicy: scope value applies only if route value is zero.
//   - metadata: scope provides base; route-level opts override per key.
func (s *Scope) applyToRoute(route *Route) {
	// Merge defaults: scope provides fallback, route-level overrides
	if s.defaults != nil {
		merged := s.defaults.Clone()
		for k, v := range route.Defaults {
			merged[k] = v
		}
		route.Defaults = merged
	}

	// Prepend scope constraints (outer-first)
	if len(s.constraints) > 0 {
		combined := make([]Constraint, 0, len(s.constraints)+len(route.Constraints))
		combined = append(combined, s.constraints...)
		combined = append(combined, route.Constraints...)
		route.Constraints = combined
	}

	// Inherit query mode if route doesn't specify
	if route.QueryMode == QueryLoose && s.queryMode != QueryLoose {
		route.QueryMode = s.queryMode
	}

	// Inherit canonical policy if route doesn't specify
	if route.CanonicalPolicy == CanonicalIgnore && s.canonicalPolicy != CanonicalIgnore {
		route.CanonicalPolicy = s.canonicalPolicy
	}

	// Merge metadata: scope provides base, route-level overrides per key
	if s.metadata != nil {
		merged := make(map[string]string, len(s.metadata))
		for k, v := range s.metadata {
			merged[k] = v
		}
		for k, v := range route.Metadata {
			merged[k] = v
		}
		route.Metadata = merged
	}
}

// effectiveName computes the fully-qualified route name by prepending the
// accumulated name prefix (with "." separator).
func (s *Scope) effectiveName(name string) string {
	if s.namePrefix == "" {
		return name
	}
	return s.namePrefix + "." + name
}

// effectiveTemplate computes the full template string by prepending the
// accumulated template prefix.
func (s *Scope) effectiveTemplate(tmpl string) string {
	return s.templatePrefix + tmpl
}
