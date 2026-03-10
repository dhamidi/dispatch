# dispatch

Semantic HTTP routing for Go.

`dispatch` provides named, reversible routes built on [URI templates](https://github.com/dhamidi/uritemplate), deterministic multi-candidate selection, post-match constraints, canonical URL handling, route scoping, and full `net/http` compatibility.

```go
r := dispatch.New()

r.GET("users.show", "/users/{id}",
    http.HandlerFunc(showUser),
    dispatch.WithConstraint(dispatch.Int("id")),
)

http.ListenAndServe(":8080", r)
```

## Installation

```sh
go get github.com/dhamidi/dispatch
```

Requires Go 1.26+. The only external dependency is [`github.com/dhamidi/uritemplate`](https://github.com/dhamidi/uritemplate).

## Quick start

Register routes by name, template, and handler. The router implements `http.Handler`.

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/dhamidi/dispatch"
)

func main() {
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

    log.Fatal(http.ListenAndServe(":8080", r))
}
```

## How-to guides

### Register routes

Use the convenience methods for common HTTP methods:

```go
r.GET("users.list", "/users", listHandler)
r.POST("users.create", "/users", createHandler)
r.PUT("users.update", "/users/{id}", updateHandler)
r.PATCH("users.patch", "/users/{id}", patchHandler)
r.DELETE("users.delete", "/users/{id}", deleteHandler)
```

Each method accepts optional `RouteOption` values:

```go
r.GET("users.show", "/users/{id}", showHandler,
    dispatch.WithConstraint(dispatch.Int("id")),
    dispatch.WithDefaults(dispatch.Params{"format": "html"}),
    dispatch.WithCanonicalPolicy(dispatch.CanonicalRedirect),
    dispatch.WithMetadata("section", "users"),
)
```

For full control, register a `Route` struct directly:

```go
r.Handle(dispatch.Route{
    Name:     "users.show",
    Methods:  dispatch.GET | dispatch.HEAD,
    Template: tmpl,
    Handler:  showHandler,
})
```

### Add constraints

Constraints validate extracted parameters after matching. The first failing constraint eliminates a candidate route.

Built-in constraints:

| Constraint | Description |
|---|---|
| `Int(key)` | Parameter parses as a base-10 integer |
| `Exact(key, value)` | Parameter equals a specific value |
| `OneOf(key, values...)` | Parameter is one of the listed values |
| `Regexp(key, re)` | Parameter matches a compiled regular expression |
| `Host(host)` | Request host matches (case-insensitive) |
| `Methods(ms)` | Request method is in the given `MethodSet` |
| `Custom(fn)` | Arbitrary predicate function |

```go
r.GET("articles.show", "/articles/{slug}",
    showArticle,
    dispatch.WithConstraint(dispatch.Regexp("slug", regexp.MustCompile(`^[a-z0-9-]+$`))),
)
```

### Group routes with scopes

Scopes share name prefixes, template prefixes, defaults, constraints, and policies across a group of routes.

```go
r.Scope(func(s *dispatch.Scope) {
    s.GET("list", "/users", listHandler)
    s.GET("show", "/users/{id}", showHandler)
    s.POST("create", "/users", createHandler)
},
    dispatch.WithNamePrefix("users"),
    dispatch.WithTemplatePrefix("/api/v1"),
)
// Registers: users.list  -> /api/v1/users
//            users.show  -> /api/v1/users/{id}
//            users.create -> /api/v1/users
```

Scopes nest. Inner values override outer values for defaults and metadata; constraints append outer-first.

```go
r.Scope(func(api *dispatch.Scope) {
    api.Scope(func(admin *dispatch.Scope) {
        admin.GET("dashboard", "/dashboard", dashHandler)
        // Registered as: api.admin.dashboard -> /api/admin/dashboard
    }, dispatch.WithNamePrefix("admin"), dispatch.WithTemplatePrefix("/admin"))
}, dispatch.WithNamePrefix("api"), dispatch.WithTemplatePrefix("/api"))
```

You can also use `WithScope` for a detached scope:

```go
api := r.WithScope(
    dispatch.WithNamePrefix("api"),
    dispatch.WithTemplatePrefix("/api/v2"),
)
api.GET("health", "/health", healthHandler)
```

### Generate URLs

Every route is reversible. Generate URLs from route names and parameters:

```go
u, err := r.URL("users.show", dispatch.Params{"id": "42"})
// u.String() == "/users/42"

path, err := r.Path("search", dispatch.Params{"q": "golang", "page": "2"})
// path == "/search?q=golang&page=2"
```

### Typed URL helpers with BindHelpers

`BindHelpers` generates type-safe URL helper functions by binding struct fields to named routes. Define a struct with `func` fields tagged with `route:"<name>"`, then call `BindHelpers` once at startup:

```go
var urls struct {
    UsersShow  func(id int64) string          `route:"users.show"`
    Search     func(q string, page int) string `route:"search"`
    PostsIndex func() string                   `route:"posts.index"`
}
r.BindHelpers(&urls)

urls.UsersShow(42)          // "/users/42"
urls.Search("golang", 1)    // "/search?q=golang&page=1"
urls.PostsIndex()           // "/posts"
```

Function arguments are matched positionally to the route's template variables (path variables first, then query variables, in declaration order). Supported argument types: `string`, `int`, `int64`, `int32`, `uint`, `uint64`, `uint32`, `float64`, `bool`, and any type implementing `fmt.Stringer`.

Return type must be `string` or `(string, error)`. Functions returning only `string` panic on generation failure; `(string, error)` functions return the error instead.

`BindHelpers` panics if the destination is not a struct pointer, a route tag references an unknown route, the argument count doesn't match the template variables, or an argument type is unsupported. Fields without a `route` tag and unexported fields are silently skipped.

### Access match data in handlers

After dispatch, route metadata is available through the request context:

```go
func showUser(w http.ResponseWriter, req *http.Request) {
    // Full match (route, params, canonical info)
    m, ok := dispatch.MatchFromContext(req.Context())

    // Just the params
    params, ok := dispatch.ParamsFromContext(req.Context())

    // Just the route name
    name, ok := dispatch.RouteNameFromContext(req.Context())
}
```

### Control query parameter handling

Set the query mode per route or as a router default:

```go
// Ignore undeclared query params (default)
r.GET("search", "/search{?q}", handler, dispatch.WithQueryMode(dispatch.QueryLoose))

// Normalize canonical form for declared params
r.GET("search", "/search{?q,page}", handler, dispatch.WithQueryMode(dispatch.QueryCanonical))

// Reject requests with undeclared query params
r.GET("search", "/search{?q}", handler, dispatch.WithQueryMode(dispatch.QueryStrict))
```

### Handle canonical URLs

Control what happens when the request URL differs from the canonical form:

```go
// Ignore differences (default)
dispatch.WithCanonicalPolicy(dispatch.CanonicalIgnore)

// Expose canonical data in Match but don't redirect
dispatch.WithCanonicalPolicy(dispatch.CanonicalAnnotate)

// Redirect to canonical URL (301 by default)
dispatch.WithCanonicalPolicy(dispatch.CanonicalRedirect)

// Reject non-canonical requests
dispatch.WithCanonicalPolicy(dispatch.CanonicalReject)
```

Set the redirect status code:

```go
r.GET("page", "/pages/{slug}", handler,
    dispatch.WithCanonicalPolicy(dispatch.CanonicalRedirect),
    dispatch.WithRedirectCode(http.StatusPermanentRedirect), // 308
)
```

## Reference

### Router options

Pass these to `dispatch.New()`:

| Option | Description | Default |
|---|---|---|
| `WithNotFoundHandler(h)` | Handler for unmatched requests | 404 text response |
| `WithMethodNotAllowedHandler(h)` | Handler for method mismatches | 405 text response |
| `WithErrorHandler(h)` | Handler for internal dispatch errors | nil |
| `WithDefaultQueryMode(m)` | Default `QueryMode` for all routes | `QueryLoose` |
| `WithDefaultCanonicalPolicy(p)` | Default `CanonicalPolicy` for all routes | `CanonicalIgnore` |
| `WithDefaultRedirectCode(code)` | Default redirect status code | 301 |
| `WithImplicitHEAD(bool)` | GET routes also match HEAD | true |

### Route options

Pass these to `r.GET(...)` and other convenience methods:

| Option | Description |
|---|---|
| `WithDefaults(Params)` | Fallback values for absent template variables |
| `WithConstraint(Constraint)` | Append a single constraint |
| `WithConstraints(...Constraint)` | Append multiple constraints |
| `WithQueryMode(QueryMode)` | Override query handling for this route |
| `WithCanonicalPolicy(CanonicalPolicy)` | Override canonical behavior for this route |
| `WithRedirectCode(int)` | HTTP status for canonical redirects |
| `WithPriority(int)` | Explicit tie-breaker (higher wins) |
| `WithMetadata(key, value)` | Attach opaque metadata |

### Errors

Registration errors:

- `ErrEmptyRouteName` — route name is empty
- `ErrDuplicateRoute` — route name already registered
- `ErrNilTemplate` — template is nil
- `ErrNilHandler` — handler is nil

Matching errors:

- `ErrNotFound` — no route matches
- `ErrMethodNotAllowed` — URL matches but method does not

Generation errors:

- `ErrUnknownRoute` — route name not found
- `ErrMissingParam` — required template variable not provided

### Introspection

```go
// Look up a single route by name
route, ok := r.Route("users.show")

// List all registered routes
routes := r.Routes()
```

### Matching semantics

When multiple routes match a request, the router selects the best candidate deterministically using these criteria (in order):

1. More literal path segments
2. More constrained parameters
3. Fewer broad/wildcard expansions
4. More declared query matches
5. Higher explicit priority
6. Earlier registration order (final tie-breaker)

## Explanation

### Why named routes?

Every route has a unique, stable name (e.g. `users.show`). Names decouple URL generation from URL structure. Change a template from `/users/{id}` to `/u/{id}` and all generated URLs update automatically — no grep-and-replace needed.

### Why URI templates?

[RFC 6570 URI templates](https://www.rfc-editor.org/rfc/rfc6570) provide a single representation that works for both matching inbound requests and generating outbound URLs. This eliminates the class of bugs where match patterns and URL builders diverge.

### Why constraints instead of regex-in-path?

Constraints are evaluated after template extraction, keeping the template clean and the validation composable. You can combine `Int("id")` with `Host("api.example.com")` without encoding either concern into the URL pattern.

### Concurrency model

Register all routes during startup, then treat the router as immutable. `ServeHTTP` is safe for concurrent use. Concurrent registration during serving is not supported.

## License

MIT. See [LICENSE](LICENSE).
