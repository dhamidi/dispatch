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
r.OPTIONS("cors.preflight", "/users", preflightHandler)
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

### Register RESTful resources

`Resource` registers standard plural resource routes following Rails conventions. Only non-nil handlers are registered:

```go
r.Resource("posts", dispatch.ResourceHandlers{
    Index:   http.HandlerFunc(listPosts),
    Show:    http.HandlerFunc(showPost),
    Create:  http.HandlerFunc(createPost),
    Update:  http.HandlerFunc(updatePost),
    Destroy: http.HandlerFunc(deletePost),
})
```

This registers the following routes:

| Method | Path | Route Name | Handler |
|---|---|---|---|
| GET | /posts | posts.index | Index |
| GET | /posts/new | posts.new | New |
| POST | /posts | posts.create | Create |
| GET | /posts/{id} | posts.show | Show |
| GET | /posts/{id}/edit | posts.edit | Edit |
| PUT, PATCH | /posts/{id} | posts.update | Update |
| DELETE | /posts/{id} | posts.destroy | Destroy |

Member routes (show, edit, update, destroy) automatically include an `Int` constraint on the ID parameter.

For a resource without a collection (no Index, no ID parameter), use `SingularResource`:

```go
r.SingularResource("account", dispatch.ResourceHandlers{
    Show:   http.HandlerFunc(showAccount),
    Update: http.HandlerFunc(updateAccount),
})
```

This registers:

| Method | Path | Route Name | Handler |
|---|---|---|---|
| GET | /account/new | account.new | New |
| POST | /account | account.create | Create |
| GET | /account | account.show | Show |
| GET | /account/edit | account.edit | Edit |
| PUT, PATCH | /account | account.update | Update |
| DELETE | /account | account.destroy | Destroy |

Customize resource registration with `ResourceOption` values:

```go
r.Resource("posts", handlers,
    dispatch.WithParamName("post_id"),  // use {post_id} instead of {id}
    dispatch.WithExcludePATCH(),        // Update matches PUT only
)
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

### Extract typed parameters in handlers

Use the `Param*` helpers to extract and convert route parameters directly from the request, instead of manually reading from the match context:

```go
func showUser(w http.ResponseWriter, req *http.Request) {
    id, err := dispatch.ParamInt(req, "id")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "user=%d", id)
}
```

Available extraction functions:

| Function | Return type | Description |
|---|---|---|
| `ParamString(r, name)` | `(string, bool)` | Raw string value; false if missing |
| `ParamInt(r, name)` | `(int, error)` | Base-10 integer |
| `ParamInt64(r, name)` | `(int64, error)` | Base-10 int64 |
| `ParamFloat64(r, name)` | `(float64, error)` | 64-bit float |
| `ParamBool(r, name)` | `(bool, error)` | Accepts true/false, 1/0, yes/no (case-insensitive) |
| `MustParamInt(r, name)` | `int` | Panics on error |
| `MustParamInt64(r, name)` | `int64` | Panics on error |
| `ParamAs(r, name, dest)` | `error` | Custom parsing via `ParamValue` interface |

When a route has a constraint that guarantees the parameter is valid (e.g. `dispatch.Int("id")`), use the `Must` variants to skip error handling:

```go
r.GET("users.show", "/users/{id}",
    http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        id := dispatch.MustParamInt(req, "id") // safe — Int constraint already validated
        fmt.Fprintf(w, "user=%d", id)
    }),
    dispatch.WithConstraint(dispatch.Int("id")),
)
```

For custom types, implement `ParamValue` (`String() string` and `Set(string) error`, mirroring `flag.Value`) and use `ParamAs`. The `String()` method enables reverse routing / URL generation via `BindHelpers`:

```go
type UserRole int

const (
    RoleAdmin UserRole = iota
    RoleEditor
)

func (r UserRole) String() string {
    switch r {
    case RoleAdmin:
        return "admin"
    case RoleEditor:
        return "editor"
    default:
        return fmt.Sprintf("unknown(%d)", int(r))
    }
}

func (r *UserRole) Set(raw string) error {
    switch raw {
    case "admin":
        *r = RoleAdmin
    case "editor":
        *r = RoleEditor
    default:
        return fmt.Errorf("unknown role %q", raw)
    }
    return nil
}

func handleRole(w http.ResponseWriter, req *http.Request) {
    var role UserRole
    if err := dispatch.ParamAs(req, "role", &role); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "role=%d", role)
}
```

Parse errors are returned as `*ParamError`, which wraps the underlying error and includes the parameter name and raw value. Missing parameters return `ErrParamNotFound`.

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

### Handle trailing slashes

Control whether the router normalizes trailing slashes by redirecting to the alternate form when it matches a registered route:

```go
r := dispatch.New(dispatch.WithDefaultSlashPolicy(dispatch.SlashRedirect))
```

When `SlashRedirect` is enabled, requests to `/users/` will 301-redirect to `/users` (or vice versa) if the alternate form matches a registered route. This is especially useful for parameterized routes — without normalization, a trailing slash can be absorbed into a path parameter (e.g., `/posts/42/` matching `{id}` as `"42/"`).

The default policy is `SlashIgnore`, which performs no trailing-slash normalization.

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
| `WithDefaultSlashPolicy(p)` | Trailing-slash normalization policy | `SlashIgnore` |
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

### Scope options

Pass these to `r.Scope(...)` or `r.WithScope(...)`:

| Option | Description |
|---|---|
| `WithNamePrefix(prefix)` | Prepend prefix to route names (separated by `.`) |
| `WithTemplatePrefix(prefix)` | Prepend prefix to URI templates |
| `WithScopeDefaults(params)` | Merge default parameters into scoped routes |
| `WithScopeConstraint(c)` | Append constraint to all scoped routes |
| `WithScopeQueryMode(qm)` | Override query mode for scoped routes |
| `WithScopeCanonicalPolicy(cp)` | Override canonical policy for scoped routes |
| `WithScopeMetadata(key, value)` | Attach metadata to all scoped routes |

Scopes expose the same convenience methods as `Router`:

- `s.GET(name, tmpl, handler, opts...)`, `s.POST(...)`, `s.PUT(...)`, `s.PATCH(...)`, `s.DELETE(...)`
- `s.Handle(route Route) error`
- `s.Scope(fn func(*Scope), opts ...ScopeOption)` — nested scoping

### Resource options

Pass these to `r.Resource(...)` and `r.SingularResource(...)`:

| Option | Description | Default |
|---|---|---|
| `WithParamName(name)` | Change the ID parameter name in member routes | `"id"` |
| `WithExcludePATCH()` | Exclude PATCH from the Update action (PUT only) | both PUT and PATCH |

### MethodSet

`MethodSet` is a bitfield representing a set of HTTP methods. Individual methods (`GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `TRACE`, `CONNECT`) are constants that can be combined with bitwise OR.

| Method / Constructor | Signature | Description |
|---|---|---|
| `Has` | `(ms MethodSet) Has(other MethodSet) bool` | Reports whether `ms` includes every method in `other` |
| `String` | `(ms MethodSet) String() string` | Pipe-separated list of methods (e.g. `"GET\|HEAD"`); empty set returns `"<none>"` |
| `MethodFromString` | `MethodFromString(method string) (MethodSet, error)` | Convert a standard HTTP method string to its `MethodSet` bit (case-sensitive) |
| `MethodSetFrom` | `MethodSetFrom(methods ...string) (MethodSet, error)` | Convert one or more method name strings to a combined `MethodSet` (case-insensitive) |

### Params

`Params` is a `map[string]string` holding route parameters.

| Method | Signature | Description |
|---|---|---|
| `Get` | `(p Params) Get(key string) string` | Returns the value for key, or `""` if missing |
| `Lookup` | `(p Params) Lookup(key string) (string, bool)` | Returns value and whether the key was present |
| `Clone` | `(p Params) Clone() Params` | Returns a shallow copy; mutations do not affect the original |

### Errors

Registration errors:

- `ErrEmptyRouteName` — route name is empty
- `ErrDuplicateRoute` — route name already registered
- `ErrNilTemplate` — template is nil
- `ErrNilHandler` — handler is nil

Matching errors:

- `ErrNotFound` — no route matches
- `ErrMethodNotAllowed` — URL matches but method does not

Method parsing errors:

- `*MethodError` — unrecognised HTTP method name (returned by `MethodSetFrom`)

Parameter extraction errors:

- `ErrParamNotFound` — named parameter not present in the match context
- `*ParamError` — parameter value could not be parsed (wraps the underlying error, includes parameter name and raw value)

Generation errors:

- `ErrUnknownRoute` — route name not found
- `ErrMissingParam` — required template variable not provided

### Introspection

```go
// Look up a single route by name
route, ok := r.Route("users.show")

// List all registered routes
routes := r.Routes()

// Match a request programmatically without dispatching
m, err := r.Match(req)
if err != nil {
    // handle ErrNotFound or ErrMethodNotAllowed
}
fmt.Println(m.Name, m.Params)
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
