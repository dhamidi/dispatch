package dispatch

import (
	"fmt"
	"strings"
)

// Handle registers a route with the router. It returns an error if the route
// fails validation. See the validation rules below.
//
// Registration is not safe for concurrent use; call Handle during startup
// before the router begins serving requests.
//
// Validation rules:
//   - route.Name must not be empty (returns ErrEmptyRouteName)
//   - route.Name must not duplicate an already-registered route (returns ErrDuplicateRoute)
//   - route.Template must not be nil (returns ErrNilTemplate)
//   - route.Handler must not be nil (returns ErrNilHandler)
//   - route.Methods must not be zero (returns a descriptive error)
//   - if route.CanonicalPolicy is CanonicalRedirect and route.RedirectCode is
//     non-zero, it must be a valid 3xx status code (returns a descriptive error)
func (r *Router) Handle(route Route) error {
	// 1. validate name
	if route.Name == "" {
		return ErrEmptyRouteName
	}
	// 2. check for duplicate name
	if _, exists := r.byName[route.Name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateRoute, route.Name)
	}
	// 3. validate template
	if route.Template == nil {
		return ErrNilTemplate
	}
	// 4. validate handler
	if route.Handler == nil {
		return ErrNilHandler
	}
	// 5. validate method set
	if route.Methods == 0 {
		return fmt.Errorf("dispatch: route %q has no methods", route.Name)
	}
	// 6. validate redirect code if applicable
	if route.CanonicalPolicy == CanonicalRedirect && route.RedirectCode != 0 {
		if route.RedirectCode < 300 || route.RedirectCode > 399 {
			return fmt.Errorf("dispatch: route %q has invalid redirect code %d", route.Name, route.RedirectCode)
		}
	}
	// 7. clone mutable maps to prevent aliasing
	if route.Defaults != nil {
		route.Defaults = route.Defaults.Clone()
	}
	if route.Metadata != nil {
		m := make(map[string]string, len(route.Metadata))
		for k, v := range route.Metadata {
			m[k] = v
		}
		route.Metadata = m
	}
	if route.Constraints != nil {
		cs := make([]Constraint, len(route.Constraints))
		copy(cs, route.Constraints)
		route.Constraints = cs
	}
	// 8. build registeredRoute with precomputed score hints
	idx := len(r.routes)
	rr := &registeredRoute{
		Route: route,
		index: idx,
		score: computeScoreHints(route, idx),
	}
	r.routes = append(r.routes, rr)
	r.byName[route.Name] = rr
	return nil
}

// MustHandle registers a route and panics if registration fails.
// It is a convenience wrapper around [Router.Handle] for use during
// package-level or init-time route setup where error handling would be
// cumbersome.
func (r *Router) MustHandle(route Route) {
	if err := r.Handle(route); err != nil {
		panic(err)
	}
}

// computeScoreHints pre-computes the structural candidateScore for a route
// at registration time to avoid repeated analysis during request matching.
func computeScoreHints(route Route, idx int) candidateScore {
	// Use the template's raw string representation to estimate segments.
	raw := route.Template.String()
	constrainedCount := len(route.Constraints)
	literalCount := countLiteralSegments(raw)
	varCount := countTemplateVars(raw)
	broadVars := varCount - constrainedCount
	if broadVars < 0 {
		broadVars = 0
	}
	return candidateScore{
		LiteralSegments: literalCount,
		ConstrainedVars: constrainedCount,
		BroadVars:       broadVars,
		Priority:        route.Priority,
		Registration:    idx,
	}
}

// countLiteralSegments counts path segments (split by '/') that contain no '{'
// characters. The query portion (after '?') is excluded.
func countLiteralSegments(raw string) int {
	// Strip query portion
	if idx := strings.IndexByte(raw, '?'); idx >= 0 {
		raw = raw[:idx]
	}
	segments := strings.Split(raw, "/")
	count := 0
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if !strings.Contains(seg, "{") {
			count++
		}
	}
	return count
}

// countTemplateVars counts the number of '{...}' expressions in the raw
// template string. The query portion is excluded to focus on path variables.
func countTemplateVars(raw string) int {
	// Strip query portion
	if idx := strings.IndexByte(raw, '?'); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.Count(raw, "{")
}
