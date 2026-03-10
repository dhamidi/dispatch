package dispatch

import (
	"errors"
	"net/http"
)

// ServeHTTP implements [http.Handler]. It resolves the request, enriches the
// request context with a [Match], and dispatches to the matched handler.
//
// Dispatch behaviour:
//   - If a route matches and its CanonicalPolicy is CanonicalRedirect and the
//     request is non-canonical, ServeHTTP issues an HTTP redirect to the
//     canonical URL.
//   - If a route matches and CanonicalPolicy is CanonicalReject and the
//     request is non-canonical, the not-found handler is invoked.
//   - Otherwise ServeHTTP enriches the request context and calls the route handler.
//   - If no route matches (ErrNotFound), the not-found handler is invoked.
//   - If a route matched structurally but the method was wrong
//     (ErrMethodNotAllowed), the method-not-allowed handler is invoked.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m, err := r.Match(req)
	switch {
	case err == nil:
		// Canonical redirect takes precedence over handler dispatch.
		if m.RedirectNeeded && m.CanonicalURL != nil {
			code := m.Route.RedirectCode
			if code == 0 {
				code = r.config.defaultRedirectCode
			}
			http.Redirect(w, req, m.CanonicalURL.String(), code)
			return
		}
		// Enrich context and dispatch.
		ctx := storeMatchInContext(req.Context(), m)
		r2 := req.WithContext(ctx)
		m.Route.Handler.ServeHTTP(w, r2)
	case isErr(err, ErrNotFound):
		r.config.notFoundHandler.ServeHTTP(w, req)
	case isErr(err, ErrMethodNotAllowed):
		r.config.methodNotAllowedHandler.ServeHTTP(w, req)
	default:
		// Unexpected internal error: fall back to 500.
		http.Error(w, "internal routing error", http.StatusInternalServerError)
	}
}

func isErr(err, target error) bool {
	return errors.Is(err, target)
}
