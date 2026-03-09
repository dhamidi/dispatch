# RFC: `dispatch` — Semantic HTTP Dispatch and Routing for Go

- Status: Draft
- Intended Audience: Library authors and application developers
- Package: `dispatch`
- Language: Go
- Dependencies: `github.com/dhamidi/uritemplate` only

## 1. Abstract

This document specifies `dispatch`, a Go package for semantic HTTP routing inspired by the practical strengths of Rails `ActionDispatch` while remaining idiomatic to Go.

`dispatch` provides:

- semantic route registration
- reversible routing built on `dhamidi/uritemplate`
- deterministic route selection
- post-match constraints
- canonical URL generation and optional redirect handling
- route grouping and scoped registration
- request context enrichment with route metadata
- plain `net/http` interoperability

`dispatch` does **not** provide:

- controller discovery
- reflection-based handler invocation
- code generation requirements
- middleware framework semantics beyond standard `net/http`
- dynamic dependencies beyond `github.com/dhamidi/uritemplate`

## 2. Goals

### 2.1 Primary Goals

The package SHALL:

1. expose an idiomatic `net/http` compatible router
2. use URI templates as the single source of truth for matching and URL generation
3. treat route identity as application-defined strings
4. support both path and query semantics through URI templates
5. provide deterministic selection when multiple routes match
6. allow route semantics to be refined with explicit constraints
7. compute canonical URLs from matched routes
8. remain dependency-light and implementation-transparent

### 2.2 Non-Goals

The package SHALL NOT:

1. emulate Rails DSL syntax
2. infer handlers from names
3. maintain a global route registry
4. require reflection for normal operation
5. require generated code for type safety
6. own sessions, cookies, rendering, or ORM behavior

## 3. Design Principles

### 3.1 Standard Library First

`dispatch` SHALL integrate directly with `net/http`.

- handlers SHALL be `http.Handler` or `http.HandlerFunc`
- router SHALL implement `http.Handler`
- context propagation SHALL use `context.Context`
- errors SHALL be returned where possible rather than hidden behind panics

### 3.2 Semantic Routing

A route is not only a URL pattern. A route is a semantic definition consisting of:

- a stable name
- a set of allowed HTTP methods
- a URI template
- zero or more defaults
- zero or more constraints
- optional metadata
- a dispatch target

### 3.3 Single Route Model for Match and Build

The same route definition SHALL be used for both inbound matching and outbound URL generation.

### 3.4 Explicitness Over Magic

The package SHALL prefer explicit route registration, explicit defaults, explicit constraints, and explicit canonicalization behavior.

## 4. Terminology

### 4.1 Route

A registered semantic endpoint definition.

### 4.2 Route Name

An application-defined string that uniquely identifies a route within a router instance.

Examples:

- `users.index`
- `users.show`
- `search`
- `admin.reports.download`

### 4.3 Template

A parsed `uritemplate.Template` used for reversible matching and URL generation.

### 4.4 Candidate

A route that preliminarily matches a request before final selection.

### 4.5 Constraint

A post-template validation rule over extracted parameters and request state.

### 4.6 Canonical URL

The normalized URL produced by re-expanding a matched route from its final resolved parameters.

### 4.7 Scope

A registration-time grouping mechanism that prefixes names, prefixes templates, attaches metadata, or injects shared behaviors.

## 5. Package Overview

The package namespace SHALL be `dispatch`.

The package SHOULD expose the following top-level concepts:

- `Router`
- `Route`
- `Match`
- `Constraint`
- `Params`
- `QueryMode`
- `CanonicalPolicy`
- `MethodSet`
- `Scope`

## 6. Core Types

## 6.1 Params

`Params` represents route parameters after extraction, default application, and normalization.

```go
type Params map[string]string
```

### 6.1.1 Requirements

- `Params` keys SHALL be case-sensitive
- values SHALL be strings
- implementations MAY provide convenience accessors for typed conversion
- `Params` SHALL be mutable only during route resolution
- callers SHOULD treat `Params` returned in `Match` as read-only

### 6.1.2 Optional Helpers

Implementations MAY provide helpers such as:

```go
func (p Params) Get(key string) string
func (p Params) Lookup(key string) (string, bool)
func (p Params) Clone() Params
```

## 6.2 MethodSet

`MethodSet` represents allowed HTTP methods for a route as a compact bitmask.

```go
type MethodSet uint16

const (
    MethodGet     MethodSet = 1 << iota // corresponds to net/http.MethodGet
    MethodHead                          // corresponds to net/http.MethodHead
    MethodPost                          // corresponds to net/http.MethodPost
    MethodPut                           // corresponds to net/http.MethodPut
    MethodPatch                         // corresponds to net/http.MethodPatch
    MethodDelete                        // corresponds to net/http.MethodDelete
    MethodOptions                       // corresponds to net/http.MethodOptions
    MethodTrace                         // corresponds to net/http.MethodTrace
    MethodConnect                       // corresponds to net/http.MethodConnect
)
```

The nine constants above correspond exactly to the method constants defined in `net/http` (`http.MethodGet`, `http.MethodHead`, etc.).  Implementors SHOULD use these `net/http` string values when converting to and from string representations.

The package SHALL provide a constructor for converting a `net/http` method string to a `MethodSet`:

```go
// MethodFromString returns the MethodSet bit for a standard HTTP method string.
// It returns 0 and an error if the method is not one of the nine standard methods.
func MethodFromString(method string) (MethodSet, error)
```

### Design Rationale

Two alternatives were evaluated:

1. **Arbitrary-method support** — represent `MethodSet` as a string set (e.g., `map[string]struct{}`).  Flexible but heap-allocated and O(method-string-length) per check.
2. **Standard-method bitmask** — represent `MethodSet` as `uint16` with one bit per standard method.  O(1) membership test, 2-byte footprint, zero allocations.

The nine methods defined by `net/http` (`GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `TRACE`, `CONNECT`) cover the complete set of methods specified in RFC 9110 plus PATCH (RFC 5789).  These are sufficient for the vast majority of HTTP API use cases.  The bitmask representation therefore provides better performance and correctness without meaningful loss of generality.

Implementations that require non-standard HTTP methods (e.g., WebDAV verbs such as `PROPFIND`) are out of scope for this specification.

### 6.2.1 Semantics

- a route MUST declare at least one method unless explicitly documented otherwise
- `HEAD` MAY be implicitly matched by `GET` if enabled by router policy
- method matching SHALL occur before or during candidate selection
- `MethodFromString` MUST be case-sensitive and MUST reject lowercase or mixed-case method strings

## 6.3 QueryMode

`QueryMode` controls how template query variables participate in matching.

```go
type QueryMode uint8

const (
    QueryLoose QueryMode = iota
    QueryCanonical
    QueryStrict
)
```

### 6.3.1 Semantics

#### `QueryLoose`

- path and declared query variables participate in matching
- undeclared query parameters SHALL be ignored for match eligibility
- undeclared query parameters MAY be preserved for canonical comparison only if configured

#### `QueryCanonical`

- path and declared query variables participate in matching
- undeclared query parameters SHALL NOT prevent matching
- canonicalization SHALL normalize declared query parameters
- undeclared query parameters MAY be dropped from canonical URLs unless configured otherwise

#### `QueryStrict`

- path and declared query variables participate in matching
- undeclared query parameters SHALL cause match rejection

## 6.4 CanonicalPolicy

`CanonicalPolicy` controls behavior when the inbound request URL differs from the canonical URL.

```go
type CanonicalPolicy uint8

const (
    CanonicalIgnore CanonicalPolicy = iota
    CanonicalAnnotate
    CanonicalRedirect
    CanonicalReject
)
```

### 6.4.1 Semantics

#### `CanonicalIgnore`

No canonical comparison is required.

#### `CanonicalAnnotate`

The router SHALL compute canonical URL information and expose it in `Match`, but SHALL NOT redirect automatically.

#### `CanonicalRedirect`

If the request matches a route but is not canonical, the router SHALL dispatch to a redirect behavior instead of the route handler.

#### `CanonicalReject`

If a request is non-canonical, the router SHALL reject it as not dispatchable.

## 6.5 Constraint

A `Constraint` refines a candidate route after URI template extraction.

```go
type Constraint interface {
    Check(*RequestContext, Params) bool
}
```

### 6.5.1 Requirements

- constraints MUST be side-effect free
- constraints MUST NOT mutate `Params`
- constraints MAY inspect request method, host, headers, or other request attributes through `RequestContext`
- constraint evaluation order SHALL be registration order
- the first failing constraint SHALL reject the candidate

### 6.5.2 Optional Alternative Form

Implementations MAY additionally accept function adapters:

```go
type ConstraintFunc func(*RequestContext, Params) bool
```

with:

```go
func (f ConstraintFunc) Check(rc *RequestContext, p Params) bool
```

## 6.6 Route

`Route` defines a semantic endpoint.

```go
type Route struct {
    Name            string
    Methods         MethodSet
    Template        *uritemplate.Template
    Handler         http.Handler

    Defaults        Params
    Constraints     []Constraint

    QueryMode       QueryMode
    CanonicalPolicy CanonicalPolicy
    RedirectCode    int
    Priority        int

    Metadata        map[string]string
}
```

### 6.6.1 Field Semantics

#### `Name`

- MUST be non-empty
- MUST be unique within a router instance
- SHOULD be stable across process restarts

#### `Methods`

- MUST identify allowed HTTP methods
- MUST NOT be zero unless an implementation explicitly supports a method-agnostic route mode

#### `Template`

- MUST be non-nil
- MUST come from `github.com/dhamidi/uritemplate`
- MUST be the sole route pattern representation

#### `Handler`

- MUST be non-nil for dispatchable routes
- MAY be nil for pure URL generation entries only if explicitly allowed by implementation

#### `Defaults`

- MAY provide fallback values for missing template variables
- SHALL be applied after extraction and before constraints
- MUST NOT override values explicitly extracted from the request

#### `Constraints`

- MAY be empty
- SHALL be evaluated after defaults are applied

#### `QueryMode`

- controls query matching strictness as defined above
- if zero-valued and no router default exists, implementations SHOULD treat it as `QueryLoose`

#### `CanonicalPolicy`

- controls canonical URL enforcement
- if zero-valued and no router default exists, implementations SHOULD treat it as `CanonicalIgnore`

#### `RedirectCode`

- used only when canonical redirect behavior is active
- SHOULD default to `http.StatusMovedPermanently` or `http.StatusPermanentRedirect` according to implementation policy

#### `Priority`

- MAY be used as an explicit tie-breaker
- higher values SHOULD win over lower values after structural scoring

#### `Metadata`

- optional application-defined string map
- router SHALL treat metadata as opaque

## 6.7 Match

`Match` represents the selected route resolution result.

```go
type Match struct {
    Route          *Route
    Name           string
    Params         Params
    Method         string

    CanonicalURL   *url.URL
    IsCanonical    bool
    RedirectNeeded bool

    score          candidateScore
}
```

### 6.7.1 Semantics

- `Route` SHALL reference the selected route
- `Name` SHALL equal `Route.Name`
- `Params` SHALL contain extracted values plus defaults
- `CanonicalURL` MAY be nil if canonical computation was disabled
- `IsCanonical` SHALL indicate canonical equivalence when canonical evaluation occurred
- `RedirectNeeded` SHALL indicate whether redirect dispatch is required by policy

## 6.8 RequestContext

`RequestContext` contains request attributes relevant to routing and constraints.

```go
type RequestContext struct {
    Request *http.Request
    URL     *url.URL
    Method  string
    Host    string
}
```

### 6.8.1 Requirements

- `Request` MUST be non-nil during match evaluation
- implementations MAY extend this structure internally
- exported API SHOULD remain minimal and stable

## 7. Router

## 7.1 Definition

```go
type Router struct {
    // implementation-defined
}
```

The router SHALL implement:

```go
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request)
```

## 7.2 Construction

The package SHOULD provide at least one constructor:

```go
func New() *Router
```

The package MAY provide option-based construction:

```go
func New(opts ...Option) *Router
```

## 7.3 Responsibilities

A router SHALL:

1. validate route registration
2. store routes in registration order
3. resolve requests to candidate routes
4. select a single best route deterministically
5. expose route match metadata through request context
6. dispatch either to the selected handler or to canonical redirect behavior
7. support outbound URL generation by route name

## 8. Registration

## 8.1 Route Registration

The router SHOULD expose:

```go
func (r *Router) Handle(route Route) error
```

It MAY additionally expose a panic-on-error helper:

```go
func (r *Router) MustHandle(route Route)
```

### 8.1.1 Registration Validation

Registration MUST fail when:

- route name is empty
- route name duplicates an existing registered route
- template is nil
- handler is nil for a dispatchable route
- method set is invalid
- redirect code is invalid when redirect policy is enabled

## 8.2 Convenience Registration Helpers

The package SHOULD provide helpers for common methods:

```go
func (r *Router) GET(name, template string, h http.Handler, opts ...RouteOption) error
func (r *Router) POST(name, template string, h http.Handler, opts ...RouteOption) error
func (r *Router) PUT(name, template string, h http.Handler, opts ...RouteOption) error
func (r *Router) PATCH(name, template string, h http.Handler, opts ...RouteOption) error
func (r *Router) DELETE(name, template string, h http.Handler, opts ...RouteOption) error
func (r *Router) OPTIONS(name, template string, h http.Handler, opts ...RouteOption) error
```

### 8.2.1 Template Parsing

Where string templates are accepted, the router SHALL parse them using `uritemplate` during registration.

## 8.3 Route Options

The package MAY provide route option helpers such as:

- `WithDefaults(Params)`
- `WithConstraint(Constraint)`
- `WithConstraints(...Constraint)`
- `WithQueryMode(QueryMode)`
- `WithCanonicalPolicy(CanonicalPolicy)`
- `WithRedirectCode(int)`
- `WithPriority(int)`
- `WithMetadata(key, value string)`

## 9. Scoping

## 9.1 Purpose

Scopes allow grouped registration with shared prefixes and policy.

## 9.2 Scope Behavior

A scope MAY define:

- name prefix
- URI template prefix
- inherited defaults
- inherited constraints
- inherited query mode
- inherited canonical policy
- inherited metadata

## 9.3 Scope API

The package SHOULD provide either or both of:

```go
func (r *Router) Scope(fn func(*Scope))
func (r *Router) WithScope(opts ...ScopeOption) *Scope
```

with:

```go
type Scope struct {
    // implementation-defined
}
```

### 9.3.1 Scope Composition

Nested scopes SHALL compose in lexical order.

For nested scopes:

- name prefixes SHALL concatenate with a separator chosen by implementation, defaulting to `.`
- template prefixes SHALL compose into a new template string before parsing or through implementation-defined template composition
- defaults and metadata SHALL merge, with inner values overriding outer values
- constraints SHALL append outer-first then inner

## 10. Matching Model

## 10.1 Overview

Request matching SHALL proceed in the following phases:

1. normalize request method and URL
2. enumerate candidate routes
3. filter by method compatibility
4. perform URI template reverse match
5. apply defaults
6. enforce query mode
7. evaluate constraints
8. score candidates
9. select best candidate
10. compute canonical URL if applicable
11. decide dispatch, redirect, or rejection

## 10.2 Method Compatibility

The router SHALL compare the request method against route methods.

### 10.2.1 HEAD Behavior

Implementations SHOULD allow a `GET` route to satisfy `HEAD` requests unless disabled.

## 10.3 URI Template Reverse Match

Reverse matching SHALL be delegated to `dhamidi/uritemplate`.

The router SHALL treat successful template reverse matching as the foundational eligibility test.

## 10.4 Defaults Application

Defaults SHALL be applied only to missing keys.

Example:

- extracted: `{id: 42}`
- defaults: `{format: html, id: 10}`
- result: `{id: 42, format: html}`

## 10.5 Query Mode Enforcement

After reverse match and defaults application, the router SHALL evaluate query behavior according to route or router query mode.

### 10.5.1 Declared Query Parameters

Declared query parameters are those represented in the route template.

### 10.5.2 Undeclared Query Parameters

- in `QueryLoose`, undeclared keys SHALL be ignored for match acceptance
- in `QueryCanonical`, undeclared keys SHALL NOT prevent matching but MAY be removed in canonical output
- in `QueryStrict`, undeclared keys SHALL reject the route

## 10.6 Constraint Evaluation

Constraints SHALL be evaluated in registration order.

The first failed constraint SHALL reject the candidate.

## 10.7 Candidate Scoring

When multiple candidates remain, the router SHALL select the best candidate using deterministic scoring.

### 10.7.1 Scoring Requirements

Scoring SHALL prefer, in order:

1. more literal specificity
2. more constrained parameters
3. fewer wildcard-like or broader expansions
4. more declared query matches
5. higher explicit priority
6. earlier registration order as final tie-breaker

### 10.7.2 Recommended Internal Score Model

Implementations SHOULD compute an internal score roughly equivalent to:

```go
type candidateScore struct {
    LiteralSegments int
    ConstrainedVars int
    BroadVars       int
    QueryMatches    int
    Priority        int
    Registration    int
}
```

Where comparison semantics are:

- larger `LiteralSegments` wins
- larger `ConstrainedVars` wins
- smaller `BroadVars` wins
- larger `QueryMatches` wins
- larger `Priority` wins
- smaller registration index wins

### 10.7.3 Determinism

Candidate selection MUST be deterministic for identical registration state and request input.

## 10.8 No Match vs Method Not Allowed

The router SHOULD distinguish:

- no route matches the URL at all
- one or more routes match structurally but not for the request method

An implementation SHOULD support emitting `405 Method Not Allowed` when appropriate.

## 11. Canonical URL Handling

## 11.1 Canonical URL Computation

If canonical handling is enabled, the router SHALL compute a canonical URL by expanding the selected route template with final resolved parameters.

Canonical comparison SHOULD consider:

- normalized path
- normalized query ordering
- normalized inclusion or exclusion of default-valued parameters according to implementation policy
- host and scheme only if configured

## 11.2 Redirect Semantics

If canonical redirect policy is active and the request URL is not canonical:

- the route handler SHALL NOT be invoked
- the router SHALL emit a redirect response using the route’s configured redirect code or router default
- the `Location` header SHALL be the canonical URL

## 11.3 Annotation Semantics

If canonical annotate policy is active:

- the request SHALL proceed to the route handler
- the match in request context SHALL expose canonical data

## 12. Outbound URL Generation

## 12.1 URL Lookup by Name

The router SHALL provide outbound URL generation by route name.

```go
func (r *Router) URL(name string, params Params) (*url.URL, error)
func (r *Router) Path(name string, params Params) (string, error)
```

## 12.2 Semantics

Outbound generation SHALL:

1. find the route by exact name
2. merge provided params with route defaults
3. require all necessary variables to be satisfied
4. expand the URI template using `uritemplate`
5. return an error if expansion fails

## 12.3 URL Generation Errors

Generation MUST fail when:

- route name is unknown
- required template variables are missing
- template expansion returns an error

## 13. Request Context Integration

## 13.1 Context Storage

After successful route selection, the router SHALL store route metadata in the request context.

The package SHOULD expose helpers:

```go
func MatchFromContext(ctx context.Context) (*Match, bool)
func RouteNameFromContext(ctx context.Context) (string, bool)
func ParamsFromContext(ctx context.Context) (Params, bool)
```

## 13.2 Request Replacement

The handler SHALL receive a cloned `*http.Request` with enriched context.

## 14. HTTP Dispatch Semantics

## 14.1 ServeHTTP Behavior

`ServeHTTP` SHALL:

1. attempt to resolve the request
2. on success, either redirect canonically or invoke the matched handler
3. on no match, invoke configurable not-found behavior
4. on method mismatch, invoke configurable method-not-allowed behavior if supported

## 14.2 Custom Error Handlers

The router SHOULD allow configuration of:

- not-found handler
- method-not-allowed handler
- internal dispatch error handler

## 15. Introspection

The package SHOULD expose read-only route introspection.

Suggested API:

```go
func (r *Router) Route(name string) (*Route, bool)
func (r *Router) Routes() []RouteInfo
```

Where `RouteInfo` is a stable read-only summary suitable for debugging or documentation.

## 16. Recommended Constraint Library

The package MAY ship common constraints.

Recommended helpers include:

```go
func Exact(key, value string) Constraint
func OneOf(key string, values ...string) Constraint
func Regexp(key string, re *regexp.Regexp) Constraint
func Int(key string) Constraint
func Host(host string) Constraint
func Methods(ms MethodSet) Constraint
func Custom(fn func(*RequestContext, Params) bool) Constraint
```

### 16.1 Dependency Rule

If a regular expression constraint is provided, it MUST use Go standard library facilities only.

No external dependency beyond `dhamidi/uritemplate` is permitted.

## 17. Registration and Dispatch Invariants

The implementation SHALL maintain the following invariants:

1. route names are unique within a router instance
2. dispatch result is deterministic for a given request and router state
3. route generation uses the same route definitions as route matching
4. defaults never override extracted values
5. constraints never mutate params
6. canonical redirect never invokes the matched route handler
7. context helpers reflect the selected route exactly

## 18. Concurrency Requirements

### 18.1 Serving

A router SHALL be safe for concurrent request handling after registration is complete.

### 18.2 Registration

Unless explicitly documented, concurrent mutation during request serving NEED NOT be supported.

The recommended model is:

- build router during startup
- treat router as immutable during serving

## 19. Error Model

The package SHOULD use explicit errors for registration and URL generation.

Recommended sentinel or typed errors include:

- `ErrDuplicateRoute`
- `ErrEmptyRouteName`
- `ErrNilTemplate`
- `ErrNilHandler`
- `ErrUnknownRoute`
- `ErrMissingParam`
- `ErrMethodNotAllowed`
- `ErrNotFound`

Implementations MAY use typed errors instead of sentinel errors.

## 20. Compliance Requirements

A conforming implementation of `dispatch` MUST:

1. depend only on Go standard library and `github.com/dhamidi/uritemplate`
2. implement `http.Handler`
3. support route names as strings
4. support reversible routing via URI templates
5. support constraints
6. support deterministic multi-candidate selection
7. support outbound generation by route name
8. support request context match metadata

A conforming implementation SHOULD:

1. support route scopes
2. support canonical URL computation
3. support method-not-allowed responses
4. support read-only introspection
5. provide convenience registration helpers

## 21. Recommended API Surface

The following API is RECOMMENDED.

```go
package dispatch

import (
    "context"
    "net/http"
    "net/url"

    "github.com/dhamidi/uritemplate"
)

type Params map[string]string

type MethodSet uint16

const (
    MethodGet     MethodSet = 1 << iota
    MethodHead
    MethodPost
    MethodPut
    MethodPatch
    MethodDelete
    MethodOptions
    MethodTrace
    MethodConnect
)

func MethodFromString(method string) (MethodSet, error)

type QueryMode uint8
const (
    QueryLoose QueryMode = iota
    QueryCanonical
    QueryStrict
)

type CanonicalPolicy uint8
const (
    CanonicalIgnore CanonicalPolicy = iota
    CanonicalAnnotate
    CanonicalRedirect
    CanonicalReject
)

type RequestContext struct {
    Request *http.Request
    URL     *url.URL
    Method  string
    Host    string
}

type Constraint interface {
    Check(*RequestContext, Params) bool
}

type ConstraintFunc func(*RequestContext, Params) bool
func (f ConstraintFunc) Check(rc *RequestContext, p Params) bool

type Route struct {
    Name            string
    Methods         MethodSet
    Template        *uritemplate.Template
    Handler         http.Handler
    Defaults        Params
    Constraints     []Constraint
    QueryMode       QueryMode
    CanonicalPolicy CanonicalPolicy
    RedirectCode    int
    Priority        int
    Metadata        map[string]string
}

type Match struct {
    Route          *Route
    Name           string
    Params         Params
    Method         string
    CanonicalURL   *url.URL
    IsCanonical    bool
    RedirectNeeded bool
}

type Router struct {
    // unexported fields
}

func New(opts ...Option) *Router
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request)
func (r *Router) Handle(route Route) error
func (r *Router) MustHandle(route Route)
func (r *Router) Route(name string) (*Route, bool)
func (r *Router) URL(name string, params Params) (*url.URL, error)
func (r *Router) Path(name string, params Params) (string, error)
func (r *Router) Match(req *http.Request) (*Match, error)

func (r *Router) GET(name, tmpl string, h http.Handler, opts ...RouteOption) error
func (r *Router) POST(name, tmpl string, h http.Handler, opts ...RouteOption) error
func (r *Router) PUT(name, tmpl string, h http.Handler, opts ...RouteOption) error
func (r *Router) PATCH(name, tmpl string, h http.Handler, opts ...RouteOption) error
func (r *Router) DELETE(name, tmpl string, h http.Handler, opts ...RouteOption) error

func MatchFromContext(ctx context.Context) (*Match, bool)
func RouteNameFromContext(ctx context.Context) (string, bool)
func ParamsFromContext(ctx context.Context) (Params, bool)
```

## 22. Example Usage

```go
r := dispatch.New()

err := r.GET("users.show", "/users/{id}", http.HandlerFunc(showUser),
    dispatch.WithConstraint(dispatch.Int("id")),
    dispatch.WithCanonicalPolicy(dispatch.CanonicalAnnotate),
)
if err != nil {
    panic(err)
}

err = r.GET("search", "/search{?q,page}", http.HandlerFunc(search),
    dispatch.WithQueryMode(dispatch.QueryCanonical),
)
if err != nil {
    panic(err)
}

http.ListenAndServe(":8080", r)
```

Handler example:

```go
func showUser(w http.ResponseWriter, req *http.Request) {
    m, ok := dispatch.MatchFromContext(req.Context())
    if !ok {
        http.Error(w, "missing route match", http.StatusInternalServerError)
        return
    }

    id := m.Params["id"]
    _, _ = w.Write([]byte("user=" + id))
}
```

Outbound generation example:

```go
u, err := r.URL("users.show", dispatch.Params{"id": "42"})
if err != nil {
    panic(err)
}
_ = u.String()
```

## 23. Implementation Notes

### 23.1 Route Storage

An implementation SHOULD maintain:

- ordered slice for candidate scanning
- map by route name for generation lookup

### 23.2 Performance

An implementation MAY precompute per-route scoring hints at registration time, including:

- literal segment count
- declared query variable count
- broad expansion count
- constraint count

### 23.3 Immutability

An implementation SHOULD clone mutable maps supplied during registration to avoid aliasing bugs.

## 24. Security Considerations

### 24.1 Untrusted Input

Route matching operates on untrusted URL input.

Implementations SHOULD:

- avoid panics from malformed URLs
- bound internal allocations where practical
- avoid mutating shared route state during request handling

### 24.2 Canonical Redirect Safety

Canonical redirect behavior SHOULD only redirect to URLs generated from the matched route and the current request context or configured router base.

Implementations MUST NOT generate open redirects from unvalidated user-controlled hosts unless explicitly configured.

## 25. Extensibility

Future versions MAY add:

- host-aware route templates
- subrouter mounting
- typed parameter decoders
- route documentation helpers
- method override policies

Such additions SHOULD preserve the core invariants in this document.

## 26. Rationale Summary

This design adopts the useful parts of an ActionDispatch-style router without importing Rails-specific magic.

It centers routing on:

- named semantic routes
- a single reversible template representation
- explicit constraints
- deterministic selection
- canonical URL handling
- standard library interoperability

This yields a routing package that is expressive, predictable, and faithful to Go’s preference for explicit composition.

