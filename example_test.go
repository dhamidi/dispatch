package dispatch_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/dhamidi/dispatch"
)

func Example_basicRouting() {
	r := dispatch.New()

	err := r.GET("users.show", "/users/{id}",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			m, ok := dispatch.MatchFromContext(req.Context())
			if !ok {
				http.Error(w, "no match", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "route=%s id=%s\n", m.Name, m.Params["id"])
		}),
	)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	r.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Print(w.Body.String())
	// Output:
	// 200
	// route=users.show id=42
}

func ExampleRouter_URL() {
	r := dispatch.New()

	if err := r.GET("users.show", "/users/{id}",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}),
	); err != nil {
		log.Fatal(err)
	}

	u, err := r.URL("users.show", dispatch.Params{"id": "42"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.String())
	// Output:
	// /users/42
}

func ExampleRouter_Path() {
	r := dispatch.New()

	if err := r.GET("search", "/search{?q,page}",
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}),
	); err != nil {
		log.Fatal(err)
	}

	path, err := r.Path("search", dispatch.Params{"q": "golang", "page": "2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(path)
	// Output:
	// /search?q=golang&page=2
}
