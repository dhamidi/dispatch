package dispatch

import "testing"

func TestMethodSet_Has(t *testing.T) {
	tests := []struct {
		name string
		ms   MethodSet
		other MethodSet
		want bool
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
