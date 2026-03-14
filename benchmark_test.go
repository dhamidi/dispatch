package dispatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/dhamidi/uritemplate"
)

// nopHandler is a minimal handler for benchmarks.
var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

// --- helpers ----------------------------------------------------------------

// buildRouterWithNRoutes creates a router with n routes. The first route is
// always "/first" (static). The last route is "/last" (static). Between them
// are a mix of static and parameterized routes.
func buildRouterWithNRoutes(n int) *Router {
	r := New()
	for i := 0; i < n; i++ {
		var tmpl string
		switch {
		case i == 0:
			tmpl = "/first"
		case i == n-1:
			tmpl = "/last"
		case i%2 == 0:
			tmpl = fmt.Sprintf("/static-%d", i)
		default:
			tmpl = fmt.Sprintf("/param-%d/{id}", i)
		}
		r.MustHandle(Route{
			Name:     fmt.Sprintf("route-%d", i),
			Methods:  GET,
			Template: uritemplate.MustParse(tmpl),
			Handler:  nopHandler,
		})
	}
	return r
}

// mustNewRequest creates a GET request or panics.
func mustNewRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	return req
}

// requestWithMatch creates a request with a Match stored in context.
func requestWithMatch(params Params) *http.Request {
	req := httptest.NewRequest("GET", "/test", nil)
	m := &Match{Params: params}
	ctx := context.WithValue(req.Context(), contextKeyMatch{}, m)
	return req.WithContext(ctx)
}

// ============================================================================
// 1. BenchmarkMatch — Route matching at various scales
// ============================================================================

func BenchmarkMatch_Static_5Routes(b *testing.B) {
	benchmarkMatchStatic(b, 5, "/first")
}

func BenchmarkMatch_Static_50Routes(b *testing.B) {
	benchmarkMatchStatic(b, 50, "/first")
}

func BenchmarkMatch_Static_200Routes(b *testing.B) {
	benchmarkMatchStatic(b, 200, "/first")
}

func BenchmarkMatch_Static_LastRoute(b *testing.B) {
	benchmarkMatchStatic(b, 50, "/last")
}

func BenchmarkMatch_Static_NotFound(b *testing.B) {
	benchmarkMatchStatic(b, 50, "/nonexistent")
}

func benchmarkMatchStatic(b *testing.B, numRoutes int, path string) {
	b.Helper()
	r := buildRouterWithNRoutes(numRoutes)
	req := mustNewRequest("GET", path)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_Parameterized_5Routes(b *testing.B) {
	benchmarkMatchParam(b, 5)
}

func BenchmarkMatch_Parameterized_50Routes(b *testing.B) {
	benchmarkMatchParam(b, 50)
}

func BenchmarkMatch_Parameterized_200Routes(b *testing.B) {
	benchmarkMatchParam(b, 200)
}

func benchmarkMatchParam(b *testing.B, numRoutes int) {
	b.Helper()
	r := buildRouterWithNRoutes(numRoutes)
	// Match a parameterized route (odd-indexed routes are parameterized)
	path := fmt.Sprintf("/param-%d/42", 1)
	req := mustNewRequest("GET", path)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_Scaling(b *testing.B) {
	for _, n := range []int{5, 10, 25, 50, 100, 200} {
		b.Run(fmt.Sprintf("routes=%d", n), func(b *testing.B) {
			r := buildRouterWithNRoutes(n)
			req := mustNewRequest("GET", "/first")
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r.Match(req)
			}
		})
	}
}

// ============================================================================
// 2. BenchmarkMatch_WithConstraints — Constraint evaluation overhead
// ============================================================================

func BenchmarkMatch_NoConstraints(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	})
	req := mustNewRequest("GET", "/users/42")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_IntConstraint(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:        "users.show",
		Methods:     GET,
		Template:    uritemplate.MustParse("/users/{id}"),
		Handler:     nopHandler,
		Constraints: []Constraint{Int("id")},
	})
	req := mustNewRequest("GET", "/users/42")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_MultipleConstraints(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "items.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/items/{id}"),
		Handler:  nopHandler,
		Constraints: []Constraint{
			Int("id"),
			Custom(func(rc *RequestContext, p Params) bool { return true }),
		},
	})
	req := mustNewRequest("GET", "/items/42")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_RegexpConstraint(b *testing.B) {
	re := regexp.MustCompile(`[a-z0-9-]+`)
	r := New()
	r.MustHandle(Route{
		Name:        "posts.show",
		Methods:     GET,
		Template:    uritemplate.MustParse("/posts/{slug}"),
		Handler:     nopHandler,
		Constraints: []Constraint{Regexp("slug", re)},
	})
	req := mustNewRequest("GET", "/posts/my-first-post")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_HostConstraint(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:        "home",
		Methods:     GET,
		Template:    uritemplate.MustParse("/"),
		Handler:     nopHandler,
		Constraints: []Constraint{Host("example.com")},
	})
	req := mustNewRequest("GET", "/")
	req.Host = "example.com"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

// ============================================================================
// 3. BenchmarkMatch_QueryModes — Query parameter handling
// ============================================================================

func BenchmarkMatch_QueryLoose(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:      "search",
		Methods:   GET,
		Template:  uritemplate.MustParse("/search{?q}"),
		Handler:   nopHandler,
		QueryMode: QueryLoose,
	})
	req := mustNewRequest("GET", "/search?q=hello&extra=ignored")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_QueryStrict(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:      "search",
		Methods:   GET,
		Template:  uritemplate.MustParse("/search{?q}"),
		Handler:   nopHandler,
		QueryMode: QueryStrict,
	})
	// Strict match — only declared params
	req := mustNewRequest("GET", "/search?q=hello")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

func BenchmarkMatch_QueryStrict_ManyParams(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:      "search",
		Methods:   GET,
		Template:  uritemplate.MustParse("/search{?q,page,size,sort,order,filter,lang,region,format,limit}"),
		Handler:   nopHandler,
		QueryMode: QueryStrict,
	})
	req := mustNewRequest("GET", "/search?q=hello&page=1&size=10&sort=name&order=asc&filter=active&lang=en&region=us&format=json&limit=100")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

// ============================================================================
// 4. BenchmarkMatch_CandidateScoring — Scoring and selection
// ============================================================================

func BenchmarkSelectBest_2Candidates(b *testing.B) {
	benchmarkSelectBest(b, 2)
}

func BenchmarkSelectBest_10Candidates(b *testing.B) {
	benchmarkSelectBest(b, 10)
}

func BenchmarkSelectBest_50Candidates(b *testing.B) {
	benchmarkSelectBest(b, 50)
}

func benchmarkSelectBest(b *testing.B, n int) {
	b.Helper()
	r := New()
	candidates := make([]*candidate, n)
	for i := 0; i < n; i++ {
		candidates[i] = &candidate{
			route: &Route{
				Name:     fmt.Sprintf("route-%d", i),
				Methods:  GET,
				Template: uritemplate.MustParse(fmt.Sprintf("/path-%d", i)),
				Handler:  nopHandler,
			},
			params: Params{"id": "42"},
			score: candidateScore{
				LiteralSegments: n - i,
				ConstrainedVars: i % 3,
				BroadVars:       i % 2,
				QueryMatches:    i % 4,
				Priority:        i,
				Registration:    i,
			},
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.selectBest(candidates)
	}
}

// BenchmarkMatch_OverlappingRoutes benchmarks scoring when multiple routes
// match the same request, requiring candidate selection.
func BenchmarkMatch_OverlappingRoutes(b *testing.B) {
	r := New()
	// Register overlapping routes: /items/{id} with varying constraints
	r.MustHandle(Route{
		Name:     "items.generic",
		Methods:  GET,
		Template: uritemplate.MustParse("/items/{id}"),
		Handler:  nopHandler,
	})
	r.MustHandle(Route{
		Name:        "items.constrained",
		Methods:     GET,
		Template:    uritemplate.MustParse("/items/{id}"),
		Handler:     nopHandler,
		Constraints: []Constraint{Int("id")},
	})
	req := mustNewRequest("GET", "/items/42")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(req)
	}
}

// ============================================================================
// 5. BenchmarkURLGeneration — Reverse routing
// ============================================================================

func BenchmarkURL_StaticRoute(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "home",
		Methods:  GET,
		Template: uritemplate.MustParse("/"),
		Handler:  nopHandler,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL("home", nil)
	}
}

func BenchmarkURL_OneParam(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	})
	params := Params{"id": "42"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL("users.show", params)
	}
}

func BenchmarkURL_MultipleParams(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "posts.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}/posts/{post_id}"),
		Handler:  nopHandler,
	})
	params := Params{"id": "42", "post_id": "99"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL("posts.show", params)
	}
}

func BenchmarkURL_WithQueryParams(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "search",
		Methods:  GET,
		Template: uritemplate.MustParse("/search{?q,page}"),
		Handler:  nopHandler,
	})
	params := Params{"q": "hello", "page": "2"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.URL("search", params)
	}
}

func BenchmarkPath_OneParam(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	})
	params := Params{"id": "42"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Path("users.show", params)
	}
}

// ============================================================================
// 6. BenchmarkParamExtraction — Parameter parsing from request context
// ============================================================================

func BenchmarkParamString(b *testing.B) {
	req := requestWithMatch(Params{"name": "alice"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParamString(req, "name")
	}
}

func BenchmarkParamInt(b *testing.B) {
	req := requestWithMatch(Params{"id": "42"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParamInt(req, "id")
	}
}

func BenchmarkParamInt64(b *testing.B) {
	req := requestWithMatch(Params{"id": "9223372036854775807"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParamInt64(req, "id")
	}
}

func BenchmarkParamFloat64(b *testing.B) {
	req := requestWithMatch(Params{"lat": "37.7749"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParamFloat64(req, "lat")
	}
}

func BenchmarkParamBool(b *testing.B) {
	req := requestWithMatch(Params{"active": "true"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParamBool(req, "active")
	}
}

type testSlug struct{ val string }

func (s *testSlug) String() string        { return s.val }
func (s *testSlug) Set(raw string) error  { s.val = raw; return nil }

func BenchmarkParamAs_CustomType(b *testing.B) {
	req := requestWithMatch(Params{"slug": "my-post"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s testSlug
		ParamAs(req, "slug", &s)
	}
}

// ============================================================================
// 7. BenchmarkServeHTTP — Full request dispatch cycle
// ============================================================================

func BenchmarkServeHTTP_StaticMatch(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "home",
		Methods:  GET,
		Template: uritemplate.MustParse("/"),
		Handler:  nopHandler,
	})
	req := mustNewRequest("GET", "/")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkServeHTTP_ParamMatch(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:        "users.show",
		Methods:     GET,
		Template:    uritemplate.MustParse("/users/{id}"),
		Handler:     nopHandler,
		Constraints: []Constraint{Int("id")},
	})
	req := mustNewRequest("GET", "/users/42")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkServeHTTP_NotFound(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "home",
		Methods:  GET,
		Template: uritemplate.MustParse("/"),
		Handler:  nopHandler,
	})
	req := mustNewRequest("GET", "/nonexistent")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkServeHTTP_MethodNotAllowed(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "home",
		Methods:  GET,
		Template: uritemplate.MustParse("/"),
		Handler:  nopHandler,
	})
	req := mustNewRequest("POST", "/")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkServeHTTP_SlashRedirect(b *testing.B) {
	r := New(WithDefaultSlashPolicy(SlashRedirect))
	r.MustHandle(Route{
		Name:     "admin",
		Methods:  GET,
		Template: uritemplate.MustParse("/admin"),
		Handler:  nopHandler,
	})
	req := mustNewRequest("GET", "/admin/")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkServeHTTP_CanonicalRedirect(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:            "users.show",
		Methods:         GET,
		Template:        uritemplate.MustParse("/users/{id}"),
		Handler:         nopHandler,
		CanonicalPolicy: CanonicalRedirect,
	})
	// Non-canonical: extra query params that differ from canonical form
	req := mustNewRequest("GET", "/users/42?extra=1")
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		r.ServeHTTP(w, req)
	}
}

// ============================================================================
// 8. BenchmarkRegistration — Route registration
// ============================================================================

func BenchmarkHandle_SimpleRoute(b *testing.B) {
	tmpl := uritemplate.MustParse("/users/{id}")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New()
		r.Handle(Route{
			Name:     "users.show",
			Methods:  GET,
			Template: tmpl,
			Handler:  nopHandler,
		})
	}
}

func BenchmarkHandle_WithOptions(b *testing.B) {
	tmpl := uritemplate.MustParse("/users/{id}")
	re := regexp.MustCompile(`[0-9]+`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New()
		r.Handle(Route{
			Name:     "users.show",
			Methods:  GET,
			Template: tmpl,
			Handler:  nopHandler,
			Defaults: Params{"format": "json"},
			Constraints: []Constraint{
				Int("id"),
				Regexp("id", re),
			},
			Metadata: map[string]string{"auth": "required"},
		})
	}
}

func BenchmarkResource_Full(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := New()
		r.Resource("posts", ResourceHandlers{
			Index:   nopHandler,
			New:     nopHandler,
			Create:  nopHandler,
			Show:    nopHandler,
			Edit:    nopHandler,
			Update:  nopHandler,
			Destroy: nopHandler,
		})
	}
}

// ============================================================================
// 9. BenchmarkBindHelpers — Type-safe URL helper generation and invocation
// ============================================================================

type benchURLs struct {
	UsersShow func(id int) string          `route:"users.show"`
	PostsShow func(uid, pid int) string    `route:"posts.show"`
}

func BenchmarkBindHelpers_Setup(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	})
	r.MustHandle(Route{
		Name:     "posts.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{uid}/posts/{pid}"),
		Handler:  nopHandler,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var urls benchURLs
		r.BindHelpers(&urls)
	}
}

func BenchmarkBindHelpers_Invoke(b *testing.B) {
	r := New()
	r.MustHandle(Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	})
	r.MustHandle(Route{
		Name:     "posts.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{uid}/posts/{pid}"),
		Handler:  nopHandler,
	})
	var urls benchURLs
	r.BindHelpers(&urls)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = urls.UsersShow(42)
		_ = urls.PostsShow(42, 99)
	}
}

// ============================================================================
// 10. BenchmarkCanonical — Canonical URL computation
// ============================================================================

func BenchmarkComputeCanonicalURL(b *testing.B) {
	route := &Route{
		Name:     "users.show",
		Methods:  GET,
		Template: uritemplate.MustParse("/users/{id}"),
		Handler:  nopHandler,
	}
	params := Params{"id": "42"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeCanonicalURL(route, params)
	}
}

func BenchmarkIsCanonicalURL_Match(b *testing.B) {
	reqURL, _ := url.Parse("/users/42?page=1&sort=name")
	canonURL, _ := url.Parse("/users/42?page=1&sort=name")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isCanonicalURL(reqURL, canonURL)
	}
}

func BenchmarkIsCanonicalURL_Mismatch(b *testing.B) {
	reqURL, _ := url.Parse("/users/42?sort=name&page=1")
	canonURL, _ := url.Parse("/users/42?page=1&sort=name")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isCanonicalURL(reqURL, canonURL)
	}
}

func BenchmarkNormalizeQuery_Small(b *testing.B) {
	raw := "page=1&sort=name"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizeQuery(raw)
	}
}

func BenchmarkNormalizeQuery_Large(b *testing.B) {
	raw := "a=1&b=2&c=3&d=4&e=5&f=6&g=7&h=8&i=9&j=10&k=11&l=12&m=13&n=14&o=15&p=16&q=17&r=18&s=19&t=20"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizeQuery(raw)
	}
}
