package dispatch

import (
	"fmt"
	"testing"

	"github.com/dhamidi/uritemplate"
)

// stringerID is a custom type implementing fmt.Stringer for testing.
type stringerID int

func (s stringerID) String() string {
	return fmt.Sprintf("sid-%d", int(s))
}

func TestBindHelpers_BasicBinding(t *testing.T) {
	r := New()
	if err := r.GET("posts.show", "/posts/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsShow func(id int64) string `route:"posts.show"`
	}
	r.BindHelpers(&urls)

	got := urls.PostsShow(42)
	if got != "/posts/42" {
		t.Errorf("expected /posts/42, got %s", got)
	}
}

func TestBindHelpers_ZeroArgFunction(t *testing.T) {
	r := New()
	if err := r.GET("posts.index", "/posts", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsIndex func() string `route:"posts.index"`
	}
	r.BindHelpers(&urls)

	got := urls.PostsIndex()
	if got != "/posts" {
		t.Errorf("expected /posts, got %s", got)
	}
}

func TestBindHelpers_MultiArgFunction(t *testing.T) {
	r := New()
	if err := r.GET("search", "/search{?q,page}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		Search func(q string, page int) string `route:"search"`
	}
	r.BindHelpers(&urls)

	got := urls.Search("golang", 1)
	if !containsSubstring(got, "q=golang") {
		t.Errorf("expected path to contain q=golang, got %s", got)
	}
	if !containsSubstring(got, "page=1") {
		t.Errorf("expected path to contain page=1, got %s", got)
	}
}

func TestBindHelpers_StringArg(t *testing.T) {
	r := New()
	if err := r.GET("posts.byslug", "/posts/{slug}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsBySlug func(slug string) string `route:"posts.byslug"`
	}
	r.BindHelpers(&urls)

	got := urls.PostsBySlug("hello-world")
	if got != "/posts/hello-world" {
		t.Errorf("expected /posts/hello-world, got %s", got)
	}
}

func TestBindHelpers_StringerArg(t *testing.T) {
	r := New()
	if err := r.GET("items.show", "/items/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		ItemsShow func(id stringerID) string `route:"items.show"`
	}
	r.BindHelpers(&urls)

	got := urls.ItemsShow(stringerID(7))
	if got != "/items/sid-7" {
		t.Errorf("expected /items/sid-7, got %s", got)
	}
}

func TestBindHelpers_ReturnStringError(t *testing.T) {
	r := New()
	if err := r.GET("posts.show", "/posts/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsShow func(id int64) (string, error) `route:"posts.show"`
	}
	r.BindHelpers(&urls)

	got, err := urls.PostsShow(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/posts/42" {
		t.Errorf("expected /posts/42, got %s", got)
	}
}

func TestBindHelpers_PanicOnMissingRoute(t *testing.T) {
	r := New()
	var urls struct {
		Missing func() string `route:"nonexistent"`
	}
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic for missing route")
		}
		msg := fmt.Sprint(rec)
		if !containsSubstring(msg, "nonexistent") {
			t.Errorf("expected panic message to mention route name, got: %s", msg)
		}
	}()
	r.BindHelpers(&urls)
}

func TestBindHelpers_PanicOnArgCountMismatch(t *testing.T) {
	r := New()
	if err := r.GET("posts.show", "/posts/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsShow func(id int64, extra string) string `route:"posts.show"`
	}
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic for arg count mismatch")
		}
	}()
	r.BindHelpers(&urls)
}

func TestBindHelpers_PanicOnUnsupportedType(t *testing.T) {
	r := New()
	if err := r.GET("posts.show", "/posts/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsShow func(id []byte) string `route:"posts.show"`
	}
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic for unsupported type")
		}
	}()
	r.BindHelpers(&urls)
}

func TestBindHelpers_PanicOnNonPointerDest(t *testing.T) {
	r := New()
	type URLs struct {
		Home func() string `route:"home"`
	}
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic for non-pointer dest")
		}
	}()
	r.BindHelpers(URLs{})
}

func TestBindHelpers_SkipUntaggedFields(t *testing.T) {
	r := New()
	if err := r.GET("home", "/", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		Home    func() string `route:"home"`
		Unbound func() string
	}
	r.BindHelpers(&urls)

	if urls.Home == nil {
		t.Error("expected Home to be bound")
	}
	if urls.Unbound != nil {
		t.Error("expected Unbound to remain nil")
	}
}

func TestBindHelpers_SkipUnexportedFields(t *testing.T) {
	r := New()
	if err := r.GET("home", "/", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		Home func() string `route:"home"`
		priv func() string `route:"home"` //nolint:unused
	}
	r.BindHelpers(&urls)

	if urls.Home == nil {
		t.Error("expected Home to be bound")
	}
}

func TestBindHelpers_IntAndUintVariants(t *testing.T) {
	r := New()
	if err := r.GET("a", "/a/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.GET("b", "/b/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.GET("c", "/c/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.GET("d", "/d/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.GET("e", "/e/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}

	var urls struct {
		A func(id int) string   `route:"a"`
		B func(id int32) string `route:"b"`
		C func(id int64) string `route:"c"`
		D func(id uint) string  `route:"d"`
		E func(id uint64) string `route:"e"`
	}
	r.BindHelpers(&urls)

	if got := urls.A(10); got != "/a/10" {
		t.Errorf("int: expected /a/10, got %s", got)
	}
	if got := urls.B(20); got != "/b/20" {
		t.Errorf("int32: expected /b/20, got %s", got)
	}
	if got := urls.C(30); got != "/c/30" {
		t.Errorf("int64: expected /c/30, got %s", got)
	}
	if got := urls.D(40); got != "/d/40" {
		t.Errorf("uint: expected /d/40, got %s", got)
	}
	if got := urls.E(50); got != "/e/50" {
		t.Errorf("uint64: expected /e/50, got %s", got)
	}
}

func TestBindHelpers_ScopedRoutes(t *testing.T) {
	r := New()
	r.Scope(func(s *Scope) {
		s.GET("admin.posts.edit", "/admin/posts/{id}/edit", noopHandler)
	})

	var urls struct {
		AdminPostsEdit func(id int64) string `route:"admin.posts.edit"`
	}
	r.BindHelpers(&urls)

	got := urls.AdminPostsEdit(42)
	if got != "/admin/posts/42/edit" {
		t.Errorf("expected /admin/posts/42/edit, got %s", got)
	}
}

// paramValueSlug is a ParamValue type for testing BindHelpers integration.
type paramValueSlug struct {
	value string
}

func (p paramValueSlug) String() string {
	return p.value
}

func (p *paramValueSlug) Set(raw string) error {
	p.value = raw
	return nil
}

func TestBindHelpers_ParamValueArg(t *testing.T) {
	r := New()
	if err := r.GET("posts.byslug", "/posts/{slug}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		PostsBySlug func(slug paramValueSlug) string `route:"posts.byslug"`
	}
	r.BindHelpers(&urls)

	slug := paramValueSlug{value: "my-post"}
	got := urls.PostsBySlug(slug)
	if got != "/posts/my-post" {
		t.Errorf("expected /posts/my-post, got %s", got)
	}
}

func TestBindHelpers_BoolAndFloat(t *testing.T) {
	r := New()
	if err := r.GET("f", "/f/{val}", noopHandler); err != nil {
		t.Fatal(err)
	}
	if err := r.GET("g", "/g/{val}", noopHandler); err != nil {
		t.Fatal(err)
	}

	var urls struct {
		F func(val bool) string    `route:"f"`
		G func(val float64) string `route:"g"`
	}
	r.BindHelpers(&urls)

	if got := urls.F(true); got != "/f/true" {
		t.Errorf("bool: expected /f/true, got %s", got)
	}
	if got := urls.G(3.14); got != "/g/3.14" {
		t.Errorf("float64: expected /g/3.14, got %s", got)
	}
}

func TestOrderedTemplateVarNames(t *testing.T) {
	tests := []struct {
		tmpl string
		want []string
	}{
		{"/posts/{id}", []string{"id"}},
		{"/posts/{id}/comments/{commentID}", []string{"id", "commentID"}},
		{"/search{?q,page}", []string{"q", "page"}},
		{"/users/{id}{?tab}", []string{"id", "tab"}},
		{"/", nil},
	}
	for _, tt := range tests {
		tmpl := mustParseTemplate(t, tt.tmpl)
		got := orderedTemplateVarNames(tmpl)
		if len(got) != len(tt.want) {
			t.Errorf("orderedTemplateVarNames(%q) = %v, want %v", tt.tmpl, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("orderedTemplateVarNames(%q)[%d] = %q, want %q", tt.tmpl, i, got[i], tt.want[i])
			}
		}
	}
}

func mustParseTemplate(t *testing.T, raw string) *uritemplate.Template {
	t.Helper()
	tmpl, err := uritemplate.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse template %q: %v", raw, err)
	}
	return tmpl
}
