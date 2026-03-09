package dispatch

import "context"

// contextKey is an unexported type used as a key in request contexts to avoid
// collisions with other packages.
type contextKey int

const (
	matchContextKey contextKey = iota
)

// MatchFromContext retrieves the Match stored in ctx by the router after
// successful route selection (§13.1).
// Returns false if no match is present.
func MatchFromContext(ctx context.Context) (*Match, bool) {
	m, ok := ctx.Value(matchContextKey).(*Match)
	return m, ok
}

// RouteNameFromContext retrieves the matched route name from ctx.
// Returns false if no match is present.
func RouteNameFromContext(ctx context.Context) (string, bool) {
	m, ok := MatchFromContext(ctx)
	if !ok {
		return "", false
	}
	return m.Name, true
}

// ParamsFromContext retrieves the matched Params from ctx.
// Returns false if no match is present.
func ParamsFromContext(ctx context.Context) (Params, bool) {
	m, ok := MatchFromContext(ctx)
	if !ok {
		return nil, false
	}
	return m.Params, true
}

// withMatch returns a copy of ctx with match stored under matchContextKey.
// Used internally by the router after route selection (§13.2).
func withMatch(ctx context.Context, m *Match) context.Context {
	return context.WithValue(ctx, matchContextKey, m)
}
