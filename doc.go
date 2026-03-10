/*
Package dispatch implements a semantic HTTP routing library for Go.

dispatch provides named, reversible routes built on URI templates
(github.com/dhamidi/uritemplate), deterministic multi-candidate selection,
post-match constraint evaluation, canonical URL generation and redirect
handling, route scoping, and standard net/http compatibility.

# Quick start

	r := dispatch.New()

	err := r.GET("users.show", "/users/{id}",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			m, _ := dispatch.MatchFromContext(req.Context())
			fmt.Fprintf(w, "user=%s", m.Params["id"])
		}),
		dispatch.WithConstraint(dispatch.Int("id")),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.ListenAndServe(":8080", r)

# Key types

[Router] is the central type. Create one with [New], register routes, then
pass it to http.ListenAndServe or any net/http-compatible server.

[Route] defines a semantic endpoint: a name, allowed methods, a URI template,
a handler, optional defaults, optional constraints, and optional metadata.

[Match] is the result of a successful route resolution. It is stored in the
request context and retrieved with [MatchFromContext].

[Params] is a map[string]string of resolved route parameters. Use
[ParamsFromContext] to access params from a handler without first obtaining the
full Match.

[Constraint] is an interface for post-extraction validation. The package ships
common constraints: [Exact], [OneOf], [Regexp], [Int], [Host], [Methods], [Custom].

[Scope] groups route registrations under a shared name prefix and template prefix.

# Route semantics

Every route has a unique Name that is stable across process restarts. Names are
used as the key for outbound URL generation via [Router.URL] and [Router.Path].

Templates are RFC 6570 URI templates parsed by github.com/dhamidi/uritemplate.
They serve as the single source of truth for both inbound matching and outbound
expansion.

Methods is a [MethodSet] bit-flag. HEAD requests are automatically satisfied
by GET routes.

Defaults provide fallback values for absent template variables. They are applied
after extraction and never override extracted values.

Constraints are evaluated in registration order after defaults. The first
failing constraint eliminates the candidate.

[QueryMode] controls how undeclared query parameters affect matching:
[QueryLoose] ignores them, [QueryCanonical] normalises canonical form,
[QueryStrict] rejects requests with undeclared parameters.

[CanonicalPolicy] controls behaviour when the request URL differs from the
canonical URL produced by template expansion: [CanonicalIgnore] (default),
[CanonicalAnnotate], [CanonicalRedirect], [CanonicalReject].

# URL generation

	u, err := r.URL("users.show", dispatch.Params{"id": "42"})
	// u.String() == "/users/42"

# Request context

After dispatch, the matched route and params are available in the handler's
request context:

	m, ok := dispatch.MatchFromContext(req.Context())
	name, ok := dispatch.RouteNameFromContext(req.Context())
	params, ok := dispatch.ParamsFromContext(req.Context())

# Errors

Registration errors: [ErrEmptyRouteName], [ErrDuplicateRoute], [ErrNilTemplate],
[ErrNilHandler].
Matching errors: [ErrNotFound], [ErrMethodNotAllowed].
Generation errors: [ErrUnknownRoute], [ErrMissingParam].
*/
package dispatch
