package dispatch

// RouteOption is a functional option that configures a [Route] before it is
// registered. RouteOptions are applied in the order they are passed to
// convenience registration methods such as [Router.GET].
type RouteOption func(*Route)

// WithDefaults sets default parameter values applied after extraction when a
// variable is absent from the matched URL. Provided defaults are merged into
// any defaults already set on the route; keys in the provided map override
// existing keys.
func WithDefaults(defaults Params) RouteOption {
	return func(r *Route) {
		if r.Defaults == nil {
			r.Defaults = make(Params, len(defaults))
		}
		for k, v := range defaults {
			r.Defaults[k] = v
		}
	}
}

// WithConstraint appends a single [Constraint] to the route's Constraints
// slice. Constraints are evaluated in the order they are appended.
func WithConstraint(c Constraint) RouteOption {
	return func(r *Route) {
		r.Constraints = append(r.Constraints, c)
	}
}

// WithConstraints appends multiple [Constraint] values to the route's
// Constraints slice in the order provided.
func WithConstraints(cs ...Constraint) RouteOption {
	return func(r *Route) {
		r.Constraints = append(r.Constraints, cs...)
	}
}

// WithQueryMode sets the [QueryMode] for the route, overriding any router
// default. See QueryLoose, QueryCanonical, and QueryStrict.
func WithQueryMode(m QueryMode) RouteOption {
	return func(r *Route) {
		r.QueryMode = m
	}
}

// WithCanonicalPolicy sets the [CanonicalPolicy] for the route, controlling
// how the router handles non-canonical request URLs. See CanonicalIgnore,
// CanonicalAnnotate, CanonicalRedirect, and CanonicalReject.
func WithCanonicalPolicy(p CanonicalPolicy) RouteOption {
	return func(r *Route) {
		r.CanonicalPolicy = p
	}
}

// WithRedirectCode sets the HTTP status code used when this route's
// CanonicalPolicy is CanonicalRedirect. The code must be a 3xx value.
// When not set, the router default is used (typically 301).
func WithRedirectCode(code int) RouteOption {
	return func(r *Route) {
		r.RedirectCode = code
	}
}

// WithPriority sets the explicit priority for the route. Higher values win
// over lower values during candidate scoring when all structural scores are
// equal.
func WithPriority(p int) RouteOption {
	return func(r *Route) {
		r.Priority = p
	}
}

// WithMetadata sets a single key-value entry in the route's Metadata map.
// Multiple calls accumulate entries; a later call with the same key overwrites
// the earlier value.
func WithMetadata(key, value string) RouteOption {
	return func(r *Route) {
		if r.Metadata == nil {
			r.Metadata = make(map[string]string)
		}
		r.Metadata[key] = value
	}
}
