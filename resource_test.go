package dispatch

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func labelHandler(label string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := ParamsFromContext(r.Context())
		fmt.Fprintf(w, "%s params=%v", label, map[string]string(params))
	})
}

func readBody(resp *http.Response) string {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

// --- Plural Resource Tests ---

func TestResource_FullCRUD(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Index:   labelHandler("index"),
		New:     labelHandler("new"),
		Create:  labelHandler("create"),
		Show:    labelHandler("show"),
		Edit:    labelHandler("edit"),
		Update:  labelHandler("update"),
		Destroy: labelHandler("destroy"),
	})
	if err != nil {
		t.Fatal(err)
	}

	routes := r.Routes()
	if len(routes) != 7 {
		t.Errorf("expected 7 routes, got %d", len(routes))
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		method string
		path   string
		status int
		prefix string
	}{
		{"GET", "/posts", 200, "index"},
		{"GET", "/posts/new", 200, "new"},
		{"POST", "/posts", 200, "create"},
		{"GET", "/posts/42", 200, "show"},
		{"GET", "/posts/42/edit", 200, "edit"},
		{"PUT", "/posts/42", 200, "update"},
		{"PATCH", "/posts/42", 200, "update"},
		{"DELETE", "/posts/42", 200, "destroy"},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, ts.URL+tt.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("%s %s: %v", tt.method, tt.path, err)
			continue
		}
		body := readBody(resp)
		if resp.StatusCode != tt.status {
			t.Errorf("%s %s: expected %d, got %d (body: %s)", tt.method, tt.path, tt.status, resp.StatusCode, body)
		}
		if !containsSubstring(body, tt.prefix) {
			t.Errorf("%s %s: expected body to contain %q, got %q", tt.method, tt.path, tt.prefix, body)
		}
	}
}

func TestResource_IntConstraint(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Index: labelHandler("index"),
		Show:  labelHandler("show"),
	})
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Non-integer ID should not match show route
	resp, _ := http.Get(ts.URL + "/posts/abc")
	if resp.StatusCode != 404 {
		t.Errorf("GET /posts/abc: expected 404, got %d", resp.StatusCode)
	}

	// Integer ID should match
	resp, _ = http.Get(ts.URL + "/posts/42")
	body := readBody(resp)
	if resp.StatusCode != 200 {
		t.Errorf("GET /posts/42: expected 200, got %d", resp.StatusCode)
	}
	if !containsSubstring(body, "show") {
		t.Errorf("GET /posts/42: expected body to contain 'show', got %q", body)
	}
}

func TestResource_SelectiveHandlers(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Index: labelHandler("index"),
		Show:  labelHandler("show"),
	})
	if err != nil {
		t.Fatal(err)
	}

	routes := r.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

func TestResource_ExcludePATCH(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Update: labelHandler("update"),
	}, WithExcludePATCH())
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// PUT should work
	req, _ := http.NewRequest("PUT", ts.URL+"/posts/1", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("PUT /posts/1: expected 200, got %d", resp.StatusCode)
	}

	// PATCH should not match (405)
	req, _ = http.NewRequest("PATCH", ts.URL+"/posts/1", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 405 {
		t.Errorf("PATCH /posts/1: expected 405, got %d", resp.StatusCode)
	}
}

func TestResource_URLGeneration(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Index: labelHandler("index"),
		Show:  labelHandler("show"),
	})
	if err != nil {
		t.Fatal(err)
	}

	path, err := r.Path("posts.index", nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/posts" {
		t.Errorf("expected /posts, got %s", path)
	}

	path, err = r.Path("posts.show", Params{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/posts/42" {
		t.Errorf("expected /posts/42, got %s", path)
	}
}

func TestResource_CustomParamName(t *testing.T) {
	r := New()
	err := r.Resource("posts", ResourceHandlers{
		Show: labelHandler("show"),
	}, WithParamName("post_id"))
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := http.Get(ts.URL + "/posts/42")
	body := readBody(resp)
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !containsSubstring(body, "post_id:42") {
		t.Errorf("expected body to contain post_id:42, got %q", body)
	}
}

// --- Singular Resource Tests ---

func TestSingularResource_FullCRUD(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		New:     labelHandler("new"),
		Create:  labelHandler("create"),
		Show:    labelHandler("show"),
		Edit:    labelHandler("edit"),
		Update:  labelHandler("update"),
		Destroy: labelHandler("destroy"),
	})
	if err != nil {
		t.Fatal(err)
	}

	routes := r.Routes()
	if len(routes) != 6 {
		t.Errorf("expected 6 routes, got %d", len(routes))
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		method string
		path   string
		status int
		prefix string
	}{
		{"GET", "/session/new", 200, "new"},
		{"POST", "/session", 200, "create"},
		{"GET", "/session", 200, "show"},
		{"GET", "/session/edit", 200, "edit"},
		{"PUT", "/session", 200, "update"},
		{"PATCH", "/session", 200, "update"},
		{"DELETE", "/session", 200, "destroy"},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, ts.URL+tt.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("%s %s: %v", tt.method, tt.path, err)
			continue
		}
		body := readBody(resp)
		if resp.StatusCode != tt.status {
			t.Errorf("%s %s: expected %d, got %d (body: %s)", tt.method, tt.path, tt.status, resp.StatusCode, body)
		}
		if !containsSubstring(body, tt.prefix) {
			t.Errorf("%s %s: expected body to contain %q, got %q", tt.method, tt.path, tt.prefix, body)
		}
	}
}

func TestSingularResource_ShowDoesNotRequireID(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		Show: labelHandler("show"),
	})
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// GET /session should return 200, not 405
	resp, err := http.Get(ts.URL + "/session")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(resp)
	if resp.StatusCode != 200 {
		t.Errorf("GET /session: expected 200, got %d (body: %s)", resp.StatusCode, body)
	}
	if !containsSubstring(body, "show") {
		t.Errorf("GET /session: expected body to contain 'show', got %q", body)
	}
}

func TestSingularResource_UpdateAndDestroy(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		Update:  labelHandler("update"),
		Destroy: labelHandler("destroy"),
	})
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// PUT /session
	req, _ := http.NewRequest("PUT", ts.URL+"/session", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("PUT /session: expected 200, got %d", resp.StatusCode)
	}

	// PATCH /session
	req, _ = http.NewRequest("PATCH", ts.URL+"/session", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("PATCH /session: expected 200, got %d", resp.StatusCode)
	}

	// DELETE /session
	req, _ = http.NewRequest("DELETE", ts.URL+"/session", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("DELETE /session: expected 200, got %d", resp.StatusCode)
	}
}

func TestSingularResource_ExcludePATCH(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		Update: labelHandler("update"),
	}, WithExcludePATCH())
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// PUT should work
	req, _ := http.NewRequest("PUT", ts.URL+"/session", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("PUT /session: expected 200, got %d", resp.StatusCode)
	}

	// PATCH should not match (405)
	req, _ = http.NewRequest("PATCH", ts.URL+"/session", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 405 {
		t.Errorf("PATCH /session: expected 405, got %d", resp.StatusCode)
	}
}

func TestSingularResource_IndexIgnored(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		Index: labelHandler("index"),
		Show:  labelHandler("show"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Index should be ignored for singular resources
	routes := r.Routes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route (show only), got %d", len(routes))
	}
}

func TestSingularResource_URLGeneration(t *testing.T) {
	r := New()
	err := r.SingularResource("session", ResourceHandlers{
		Show: labelHandler("show"),
		New:  labelHandler("new"),
	})
	if err != nil {
		t.Fatal(err)
	}

	path, err := r.Path("session.show", nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/session" {
		t.Errorf("expected /session, got %s", path)
	}

	path, err = r.Path("session.new", nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/session/new" {
		t.Errorf("expected /session/new, got %s", path)
	}
}
