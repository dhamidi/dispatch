package dispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURL_WithDefaults(t *testing.T) {
	r := New()
	r.GET("page", "/page/{name}", noopHandler, WithDefaults(Params{"name": "home"}))
	u, err := r.URL("page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/page/home" {
		t.Errorf("expected /page/home, got %s", u.Path)
	}
}

func TestRoutes_Introspection(t *testing.T) {
	r := New()
	r.GET("a", "/a", noopHandler, WithMetadata("env", "test"))
	r.POST("b", "/b", noopHandler)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Name != "a" || routes[1].Name != "b" {
		t.Error("unexpected route names")
	}
	if routes[0].Metadata["env"] != "test" {
		t.Error("expected metadata")
	}
}

func TestComputeCanonicalURL_MatchingURL(t *testing.T) {
	// Test the isCanonical=true path by requesting a canonical URL
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, _ := MatchFromContext(req.Context())
		if m.CanonicalURL == nil {
			t.Error("expected canonical URL to be computed")
		}
		if !m.IsCanonical {
			t.Error("expected IsCanonical=true")
		}
	})
	r.GET("items", "/items/{id}", h, WithCanonicalPolicy(CanonicalAnnotate))

	req := httptest.NewRequest("GET", "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
}

func TestHelperArgToString_Uint32(t *testing.T) {
	r := New()
	if err := r.GET("u32", "/u/{id}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		U func(id uint32) string `route:"u32"`
	}
	r.BindHelpers(&urls)

	if got := urls.U(123); got != "/u/123" {
		t.Errorf("expected /u/123, got %s", got)
	}
}

func TestHelperArgToString_Float32(t *testing.T) {
	r := New()
	if err := r.GET("f32", "/f/{val}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		F func(val float32) string `route:"f32"`
	}
	r.BindHelpers(&urls)

	got := urls.F(1.5)
	if got != "/f/1.5" {
		t.Errorf("expected /f/1.5, got %s", got)
	}
}

func TestHelperArgToString_Int8(t *testing.T) {
	r := New()
	if err := r.GET("i8", "/i/{val}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		I func(val int8) string `route:"i8"`
	}
	r.BindHelpers(&urls)

	if got := urls.I(7); got != "/i/7" {
		t.Errorf("expected /i/7, got %s", got)
	}
}

func TestHelperArgToString_Uint8(t *testing.T) {
	r := New()
	if err := r.GET("u8", "/u8/{val}", noopHandler); err != nil {
		t.Fatal(err)
	}
	var urls struct {
		U func(val uint8) string `route:"u8"`
	}
	r.BindHelpers(&urls)

	if got := urls.U(255); got != "/u8/255" {
		t.Errorf("expected /u8/255, got %s", got)
	}
}

func TestResource_OnlyShowAndDestroy(t *testing.T) {
	r := New()
	err := r.Resource("things", ResourceHandlers{
		Show:    noopHandler,
		Destroy: noopHandler,
	})
	if err != nil {
		t.Fatal(err)
	}
	// show should match
	req := httptest.NewRequest("GET", "/things/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for show, got %d", rec.Code)
	}
	// destroy should match
	req = httptest.NewRequest("DELETE", "/things/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for destroy, got %d", rec.Code)
	}
	// index should not exist
	req = httptest.NewRequest("GET", "/things", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for index, got %d", rec.Code)
	}
}

func TestSingularResource_AllHandlers(t *testing.T) {
	r := New()
	err := r.SingularResource("profile", ResourceHandlers{
		Index:   noopHandler, // should be ignored for singular
		New:     noopHandler,
		Create:  noopHandler,
		Show:    noopHandler,
		Edit:    noopHandler,
		Update:  noopHandler,
		Destroy: noopHandler,
	})
	if err != nil {
		t.Fatal(err)
	}

	// show
	req := httptest.NewRequest("GET", "/profile", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// edit
	req = httptest.NewRequest("GET", "/profile/edit", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for edit, got %d", rec.Code)
	}

	// update via PUT
	req = httptest.NewRequest("PUT", "/profile", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for PUT update, got %d", rec.Code)
	}

	// update via PATCH
	req = httptest.NewRequest("PATCH", "/profile", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for PATCH update, got %d", rec.Code)
	}

	// destroy
	req = httptest.NewRequest("DELETE", "/profile", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for destroy, got %d", rec.Code)
	}

	// new
	req = httptest.NewRequest("GET", "/profile/new", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for new, got %d", rec.Code)
	}

	// create
	req = httptest.NewRequest("POST", "/profile", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for create, got %d", rec.Code)
	}
}

func TestResource_AllNilHandlers(t *testing.T) {
	r := New()
	err := r.Resource("empty", ResourceHandlers{})
	if err != nil {
		t.Fatal(err)
	}
	// No routes should be registered
	routes := r.Routes()
	for _, ri := range routes {
		if containsSubstring(ri.Name, "empty") {
			t.Errorf("unexpected route %q registered", ri.Name)
		}
	}
}

func TestResource_DuplicateNameError(t *testing.T) {
	r := New()
	// Register a resource, then try to register with same name to trigger error path
	r.GET("items.index", "/existing", noopHandler)
	err := r.Resource("items", ResourceHandlers{
		Index: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate route name")
	}
}

func TestResource_ShowError(t *testing.T) {
	r := New()
	r.GET("widgets.show", "/existing2", noopHandler)
	err := r.Resource("widgets", ResourceHandlers{
		Show: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate show route name")
	}
}

func TestResource_NewError(t *testing.T) {
	r := New()
	r.GET("gadgets.new", "/existing3", noopHandler)
	err := r.Resource("gadgets", ResourceHandlers{
		New: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate new route name")
	}
}

func TestResource_CreateError(t *testing.T) {
	r := New()
	r.POST("gizmos.create", "/existing4", noopHandler)
	err := r.Resource("gizmos", ResourceHandlers{
		Create: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate create route name")
	}
}

func TestResource_EditError(t *testing.T) {
	r := New()
	r.GET("doodads.edit", "/existing5", noopHandler)
	err := r.Resource("doodads", ResourceHandlers{
		Edit: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate edit route name")
	}
}

func TestResource_UpdateError(t *testing.T) {
	r := New()
	r.PUT("thingies.update", "/existing6", noopHandler)
	err := r.Resource("thingies", ResourceHandlers{
		Update: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate update route name")
	}
}

func TestResource_DestroyError(t *testing.T) {
	r := New()
	r.DELETE("doohickeys.destroy", "/existing7", noopHandler)
	err := r.Resource("doohickeys", ResourceHandlers{
		Destroy: noopHandler,
	})
	if err == nil {
		t.Error("expected error for duplicate destroy route name")
	}
}

func TestSingularResource_NewError(t *testing.T) {
	r := New()
	r.GET("sprocket.new", "/existing10", noopHandler)
	err := r.SingularResource("sprocket", ResourceHandlers{
		New: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_CreateError(t *testing.T) {
	r := New()
	r.POST("cog.create", "/existing11", noopHandler)
	err := r.SingularResource("cog", ResourceHandlers{
		Create: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_ShowError(t *testing.T) {
	r := New()
	r.GET("bolt.show", "/existing12", noopHandler)
	err := r.SingularResource("bolt", ResourceHandlers{
		Show: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_EditError(t *testing.T) {
	r := New()
	r.GET("nut.edit", "/existing13", noopHandler)
	err := r.SingularResource("nut", ResourceHandlers{
		Edit: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_UpdateError(t *testing.T) {
	r := New()
	r.PUT("washer.update", "/existing14", noopHandler)
	err := r.SingularResource("washer", ResourceHandlers{
		Update: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_DestroyError(t *testing.T) {
	r := New()
	r.DELETE("rivet.destroy", "/existing15", noopHandler)
	err := r.SingularResource("rivet", ResourceHandlers{
		Destroy: noopHandler,
	})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSingularResource_NilHandlers(t *testing.T) {
	r := New()
	err := r.SingularResource("nothing", ResourceHandlers{})
	if err != nil {
		t.Fatal(err)
	}
	routes := r.Routes()
	for _, ri := range routes {
		if containsSubstring(ri.Name, "nothing") {
			t.Errorf("unexpected route %q registered", ri.Name)
		}
	}
}

func TestSingularResource_ExcludePATCH_Verify(t *testing.T) {
	r := New()
	err := r.SingularResource("settings", ResourceHandlers{
		Show:   noopHandler,
		Update: noopHandler,
	}, WithExcludePATCH())
	if err != nil {
		t.Fatal(err)
	}

	// PUT should work
	req := httptest.NewRequest("PUT", "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for PUT, got %d", rec.Code)
	}

	// PATCH should not work
	req = httptest.NewRequest("PATCH", "/settings", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PATCH, got %d", rec.Code)
	}
}
