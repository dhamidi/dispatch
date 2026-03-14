package dispatch

import "testing"

func TestMethodSet_Has(t *testing.T) {
	tests := []struct {
		name  string
		ms    MethodSet
		other MethodSet
		want  bool
	}{
		{"GET has GET", GET, GET, true},
		{"GET|POST has GET", GET | POST, GET, true},
		{"GET|POST has POST", GET | POST, POST, true},
		{"GET has POST", GET, POST, false},
		{"empty has GET", MethodSet(0), GET, false},
		{"GET|POST|DELETE has GET|DELETE", GET | POST | DELETE, GET | DELETE, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ms.Has(tt.other); got != tt.want {
				t.Errorf("%s.Has(%s) = %v, want %v", tt.ms, tt.other, got, tt.want)
			}
		})
	}
}

func TestMethodSet_String(t *testing.T) {
	tests := []struct {
		name string
		ms   MethodSet
		want string
	}{
		{"single GET", GET, "GET"},
		{"GET|POST", GET | POST, "GET|POST"},
		{"empty", MethodSet(0), "<none>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ms.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMethodSet_Constants(t *testing.T) {
	all := []struct {
		name string
		val  MethodSet
	}{
		{"GET", GET},
		{"HEAD", HEAD},
		{"POST", POST},
		{"PUT", PUT},
		{"PATCH", PATCH},
		{"DELETE", DELETE},
		{"OPTIONS", OPTIONS},
		{"TRACE", TRACE},
		{"CONNECT", CONNECT},
	}

	t.Run("non-zero", func(t *testing.T) {
		for _, c := range all {
			if c.val == 0 {
				t.Errorf("%s is zero", c.name)
			}
		}
	})

	t.Run("distinct bits", func(t *testing.T) {
		for i := 0; i < len(all); i++ {
			for j := i + 1; j < len(all); j++ {
				if all[i].val&all[j].val != 0 {
					t.Errorf("%s and %s share bits", all[i].name, all[j].name)
				}
			}
		}
	})
}

func TestMethodSetFrom_Valid(t *testing.T) {
	ms, err := MethodSetFrom("GET", "POST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.Has(GET) || !ms.Has(POST) {
		t.Error("expected GET and POST in set")
	}
	if ms.Has(DELETE) {
		t.Error("should not contain DELETE")
	}
}

func TestMethodSetFrom_Invalid(t *testing.T) {
	_, err := MethodSetFrom("INVALID")
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
	me, ok := err.(*MethodError)
	if !ok {
		t.Fatalf("expected *MethodError, got %T", err)
	}
	if me.Method != "INVALID" {
		t.Errorf("expected Method=INVALID, got %s", me.Method)
	}
}

func TestMethodError_Error(t *testing.T) {
	e := &MethodError{Method: "FOO"}
	want := "unrecognised HTTP method: FOO"
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMethodFromString_AllMethods(t *testing.T) {
	methods := []struct {
		name string
		want MethodSet
	}{
		{"TRACE", TRACE},
		{"CONNECT", CONNECT},
		{"OPTIONS", OPTIONS},
		{"PUT", PUT},
		{"PATCH", PATCH},
		{"DELETE", DELETE},
		{"HEAD", HEAD},
	}
	for _, tt := range methods {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MethodFromString(tt.name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMethodSet_String_HighBit(t *testing.T) {
	// A MethodSet with bits beyond the 9 standard methods
	ms := MethodSet(1 << 10)
	got := ms.String()
	if got != "<none>" {
		t.Errorf("expected <none> for high bit, got %q", got)
	}
}

func TestMethodSetFrom_CaseInsensitive(t *testing.T) {
	ms, err := MethodSetFrom("get", "post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.Has(GET) || !ms.Has(POST) {
		t.Error("expected GET and POST (case-insensitive)")
	}
}
