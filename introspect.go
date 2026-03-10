package dispatch

// RouteInfo is a read-only summary of a registered [Route]. It is returned by
// [Router.Routes] for introspection. The Handler field is intentionally
// omitted to discourage misuse; use [Router.Route] to obtain the full Route.
type RouteInfo struct {
	// Name is the route's unique name.
	Name string

	// Methods is the set of HTTP methods the route accepts.
	Methods MethodSet

	// Template is the raw URI template string.
	Template string

	// Priority is the route's explicit priority.
	Priority int

	// Metadata is a copy of the route's metadata map.
	Metadata map[string]string
}

// Route returns the registered [Route] with the given name and true, or nil
// and false if no route with that name has been registered.
//
// The returned *Route SHOULD be treated as read-only. Mutating it may cause
// undefined behaviour during request handling.
func (r *Router) Route(name string) (*Route, bool) {
	rr, ok := r.byName[name]
	if !ok {
		return nil, false
	}
	route := rr.Route // copy
	return &route, true
}

// Routes returns a snapshot of all registered routes as [RouteInfo] values in
// registration order. The slice is a copy; modifications do not affect the
// router.
func (r *Router) Routes() []RouteInfo {
	out := make([]RouteInfo, len(r.routes))
	for i, rr := range r.routes {
		md := make(map[string]string, len(rr.Metadata))
		for k, v := range rr.Metadata {
			md[k] = v
		}
		tmplStr := ""
		if rr.Template != nil {
			tmplStr = rr.Template.String()
		}
		out[i] = RouteInfo{
			Name:     rr.Name,
			Methods:  rr.Methods,
			Template: tmplStr,
			Priority: rr.Priority,
			Metadata: md,
		}
	}
	return out
}
