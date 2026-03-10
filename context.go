package dispatch

import "context"

// contextKeyMatch is the key type used to store a *Match in a context.Context.
// Using a named unexported type prevents key collisions with other packages.
type contextKeyMatch struct{}

// MatchFromContext retrieves the [Match] stored by the router in ctx after a
// successful route resolution. It returns the Match and true if present, or
// nil and false if ctx does not contain a Match (e.g., the handler was called
// outside of a dispatch cycle).
func MatchFromContext(ctx context.Context) (*Match, bool) {
	m, ok := ctx.Value(contextKeyMatch{}).(*Match)
	return m, ok
}

// RouteNameFromContext returns the route name from ctx. It is a convenience
// wrapper around [MatchFromContext] for the common case of needing only the
// route name.
func RouteNameFromContext(ctx context.Context) (string, bool) {
	m, ok := MatchFromContext(ctx)
	if !ok {
		return "", false
	}
	return m.Name, true
}

// ParamsFromContext returns the resolved route [Params] from ctx. It is a
// convenience wrapper around [MatchFromContext].
func ParamsFromContext(ctx context.Context) (Params, bool) {
	m, ok := MatchFromContext(ctx)
	if !ok {
		return nil, false
	}
	return m.Params, true
}

// storeMatchInContext returns a copy of ctx with match stored under contextKeyMatch.
// Used internally by the router after route selection.
func storeMatchInContext(ctx context.Context, m *Match) context.Context {
	return context.WithValue(ctx, contextKeyMatch{}, m)
}
