package dispatch

import (
	"fmt"
	"net/url"

	"github.com/dhamidi/uritemplate"
)

// URL generates an absolute *url.URL for the named route by expanding its
// URI template with the provided params merged with route defaults.
//
// Params provided by the caller take precedence over route defaults.
//
// Errors:
//   - [ErrUnknownRoute]: no route with the given name is registered.
//   - [ErrMissingParam]: a required template variable is absent from params
//     and the route defaults.
//   - Other errors: the uritemplate expansion failed for a different reason.
func (r *Router) URL(name string, params Params) (*url.URL, error) {
	rr, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownRoute, name)
	}
	merged := mergeParams(rr.Defaults, params)
	expanded, err := expandTemplate(rr.Template, merged)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(expanded)
	if err != nil {
		return nil, fmt.Errorf("dispatch: invalid expanded URL %q: %w", expanded, err)
	}
	return u, nil
}

// Path generates the path-and-query string for the named route. It is a
// convenience wrapper around [Router.URL] that returns only the path portion.
//
// Returns the same errors as [Router.URL].
func (r *Router) Path(name string, params Params) (string, error) {
	u, err := r.URL(name, params)
	if err != nil {
		return "", err
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery, nil
	}
	return u.Path, nil
}

// mergeParams returns a new Params that contains all entries from defaults
// overridden by any entries in provided.
func mergeParams(defaults, provided Params) Params {
	out := make(Params, len(defaults)+len(provided))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range provided {
		out[k] = v
	}
	return out
}

// expandTemplate expands tmpl with the given params and returns the result.
// Returns ErrMissingParam if a required variable cannot be satisfied.
func expandTemplate(tmpl *uritemplate.Template, params Params) (string, error) {
	values := make(uritemplate.Values, len(params))
	for k, v := range params {
		values[k] = uritemplate.String(v)
	}
	result, err := tmpl.Expand(values)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMissingParam, err)
	}
	return result, nil
}
