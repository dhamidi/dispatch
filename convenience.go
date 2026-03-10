package dispatch

import (
	"fmt"
	"net/http"

	"github.com/dhamidi/uritemplate"
)

// registerMethod is a shared implementation for the HTTP method convenience
// helpers (GET, POST, PUT, PATCH, DELETE, OPTIONS). It parses the URI template,
// applies any RouteOptions, and delegates to Handle.
func (r *Router) registerMethod(method MethodSet, name, tmpl string, h http.Handler, opts []RouteOption) error {
	t, err := uritemplate.Parse(tmpl)
	if err != nil {
		return fmt.Errorf("dispatch: invalid template %q: %w", tmpl, err)
	}
	route := Route{
		Name:     name,
		Methods:  method,
		Template: t,
		Handler:  h,
	}
	for _, o := range opts {
		o(&route)
	}
	return r.Handle(route)
}

// GET registers a route that matches HTTP GET (and HEAD) requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) GET(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(GET, name, tmpl, h, opts)
}

// POST registers a route that matches HTTP POST requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) POST(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(POST, name, tmpl, h, opts)
}

// PUT registers a route that matches HTTP PUT requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) PUT(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(PUT, name, tmpl, h, opts)
}

// PATCH registers a route that matches HTTP PATCH requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) PATCH(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(PATCH, name, tmpl, h, opts)
}

// DELETE registers a route that matches HTTP DELETE requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) DELETE(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(DELETE, name, tmpl, h, opts)
}

// OPTIONS registers a route that matches HTTP OPTIONS requests.
//
// The template string is parsed using github.com/dhamidi/uritemplate.
// RouteOptions are applied in order after the route is constructed.
func (r *Router) OPTIONS(name, tmpl string, h http.Handler, opts ...RouteOption) error {
	return r.registerMethod(OPTIONS, name, tmpl, h, opts)
}
