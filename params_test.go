package dispatch

import "testing"

func TestParams_Get(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		p := Params{"x": "1"}
		if got := p.Get("x"); got != "1" {
			t.Errorf("Get(x) = %q, want %q", got, "1")
		}
	})
	t.Run("missing key", func(t *testing.T) {
		p := Params{"x": "1"}
		if got := p.Get("y"); got != "" {
			t.Errorf("Get(y) = %q, want %q", got, "")
		}
	})
	t.Run("nil map", func(t *testing.T) {
		p := Params(nil)
		if got := p.Get("x"); got != "" {
			t.Errorf("Get(x) on nil = %q, want %q", got, "")
		}
	})
}

func TestParams_Lookup(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		p := Params{"x": "1"}
		v, ok := p.Lookup("x")
		if v != "1" || !ok {
			t.Errorf("Lookup(x) = (%q, %v), want (%q, true)", v, ok, "1")
		}
	})
	t.Run("existing key empty value", func(t *testing.T) {
		p := Params{"x": ""}
		v, ok := p.Lookup("x")
		if v != "" || !ok {
			t.Errorf("Lookup(x) = (%q, %v), want (%q, true)", v, ok, "")
		}
	})
	t.Run("missing key", func(t *testing.T) {
		p := Params{"x": "1"}
		v, ok := p.Lookup("y")
		if v != "" || ok {
			t.Errorf("Lookup(y) = (%q, %v), want (%q, false)", v, ok, "")
		}
	})
}

func TestParams_Clone(t *testing.T) {
	t.Run("non-nil map", func(t *testing.T) {
		orig := Params{"a": "1", "b": "2"}
		c := orig.Clone()
		if len(c) != len(orig) {
			t.Fatalf("Clone length = %d, want %d", len(c), len(orig))
		}
		for k, v := range orig {
			if c[k] != v {
				t.Errorf("Clone[%q] = %q, want %q", k, c[k], v)
			}
		}
	})
	t.Run("mutation isolation", func(t *testing.T) {
		orig := Params{"a": "1"}
		c := orig.Clone()
		c["a"] = "changed"
		if orig["a"] != "1" {
			t.Errorf("original mutated: a = %q, want %q", orig["a"], "1")
		}
	})
	t.Run("nil map", func(t *testing.T) {
		p := Params(nil)
		c := p.Clone()
		if c != nil {
			t.Errorf("Clone of nil = %v, want nil", c)
		}
	})
	t.Run("empty map", func(t *testing.T) {
		p := Params{}
		c := p.Clone()
		if c == nil {
			t.Fatal("Clone of empty map is nil, want non-nil")
		}
		if len(c) != 0 {
			t.Errorf("Clone of empty map has length %d, want 0", len(c))
		}
	})
}
