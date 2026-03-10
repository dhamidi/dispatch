package dispatch_test

import (
	"fmt"
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
