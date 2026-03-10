package dispatch

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
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
//
// When the router's SlashPolicy is SlashRedirect, trailing-slash normalization
// is applied before standard matching. See [SlashRedirect] for details.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.config.defaultSlashPolicy == SlashRedirect {
		if r.serveWithSlashRedirect(w, req) {
			return
		}
	} else {
		r.serveNormal(w, req)
	}
}

// serveNormal handles the request without slash normalization.
func (r *Router) serveNormal(w http.ResponseWriter, req *http.Request) {
	m, err := r.Match(req)
	r.dispatch(w, req, m, err)
}

// dispatch handles the result of a Match call.
func (r *Router) dispatch(w http.ResponseWriter, req *http.Request, m *Match, err error) {
	switch {
	case err == nil:
		if m.RedirectNeeded && m.CanonicalURL != nil {
			code := m.Route.RedirectCode
			if code == 0 {
				code = r.config.defaultRedirectCode
			}
			http.Redirect(w, req, m.CanonicalURL.String(), code)
			return
		}
		ctx := storeMatchInContext(req.Context(), m)
		r2 := req.WithContext(ctx)
		m.Route.Handler.ServeHTTP(w, r2)
	case isErr(err, ErrNotFound):
		r.config.notFoundHandler.ServeHTTP(w, req)
	case isErr(err, ErrMethodNotAllowed):
		r.config.methodNotAllowedHandler.ServeHTTP(w, req)
	default:
		http.Error(w, "internal routing error", http.StatusInternalServerError)
	}
}

// serveWithSlashRedirect implements trailing-slash normalization.
// It returns true if it handled the request (either dispatched or redirected),
// false if it fell through to a 404/405.
func (r *Router) serveWithSlashRedirect(w http.ResponseWriter, req *http.Request) bool {
	path := req.URL.Path
	hasTrailingSlash := path != "/" && strings.HasSuffix(path, "/")

	// Phase 1: If the request has a trailing slash, try without it first.
	// This prevents parameterized routes from absorbing the trailing slash
	// into a path parameter (e.g., /posts/42/ matching {id} as "42/").
	if hasTrailingSlash {
		stripped := strings.TrimRight(path, "/")
		if stripped == "" {
			stripped = "/"
		}
		strippedReq := cloneRequestWithPath(req, stripped)
		_, err := r.Match(strippedReq)
		if err == nil {
			// The path without trailing slash matches — redirect there.
			redirectURL := buildRedirectURL(stripped, req.URL.RawQuery)
			code := r.config.defaultRedirectCode
			http.Redirect(w, req, redirectURL, code)
			return true
		}
	}

	// Phase 2: Try normal match.
	m, err := r.Match(req)
	if err == nil {
		r.dispatch(w, req, m, nil)
		return true
	}

	// Phase 3: If no match and no trailing slash, try with trailing slash.
	if isErr(err, ErrNotFound) && !hasTrailingSlash {
		withSlash := path + "/"
		withSlashReq := cloneRequestWithPath(req, withSlash)
		_, altErr := r.Match(withSlashReq)
		if altErr == nil {
			redirectURL := buildRedirectURL(withSlash, req.URL.RawQuery)
			code := r.config.defaultRedirectCode
			http.Redirect(w, req, redirectURL, code)
			return true
		}
	}

	// Fall through to normal error handling.
	r.dispatch(w, req, nil, err)
	return true
}

// cloneRequestWithPath creates a shallow clone of req with an altered URL path
// and the query string stripped. This is used for slash-probing where we only
// want to test whether the alternate path matches structurally. Query strings
// are stripped because literal templates (e.g., /admin/) cannot match a
// RequestURI that includes query parameters they don't declare.
func cloneRequestWithPath(req *http.Request, newPath string) *http.Request {
	r2 := req.Clone(req.Context())
	r2.URL = cloneURL(req.URL)
	r2.URL.Path = newPath
	r2.URL.RawPath = ""
	r2.URL.RawQuery = ""
	r2.URL.Fragment = ""
	r2.RequestURI = newPath
	return r2
}

// buildRedirectURL constructs a redirect target from a path and raw query.
func buildRedirectURL(path, rawQuery string) string {
	if rawQuery != "" {
		return path + "?" + rawQuery
	}
	return path
}

// cloneURL returns a shallow copy of u.
func cloneURL(u *url.URL) *url.URL {
	u2 := *u
	return &u2
}

func isErr(err, target error) bool {
	return errors.Is(err, target)
}
