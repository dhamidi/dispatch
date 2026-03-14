package dispatch

import (
	"testing"
)

func TestValidateHelperReturnType_IntReturn(t *testing.T) {
	r := New()
	if err := r.GET("h1", "/", noopHandler); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for int return type")
		}
	}()

	var urls struct {
		Home func() int `route:"h1"`
	}
	r.BindHelpers(&urls)
}

func TestValidateHelperReturnType_ThreeReturns(t *testing.T) {
	r := New()
	if err := r.GET("h2", "/h2", noopHandler); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for 3 return values")
		}
	}()

	var urls struct {
		Home func() (string, error, int) `route:"h2"`
	}
	r.BindHelpers(&urls)
}

func TestValidateHelperReturnType_WrongSecondReturn(t *testing.T) {
	r := New()
	if err := r.GET("h3", "/h3", noopHandler); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-error second return")
		}
	}()

	var urls struct {
		Home func() (string, string) `route:"h3"`
	}
	r.BindHelpers(&urls)
}

func TestValidateHelperReturnType_NonFuncField(t *testing.T) {
	r := New()
	if err := r.GET("h4", "/h4", noopHandler); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-func field")
		}
	}()

	var urls struct {
		Home int `route:"h4"`
	}
	r.BindHelpers(&urls)
}

func TestValidateHelperReturnType_WrongFirstReturnInPair(t *testing.T) {
	r := New()
	if err := r.GET("h5", "/h5", noopHandler); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-string first return")
		}
	}()

	var urls struct {
		Home func() (int, error) `route:"h5"`
	}
	r.BindHelpers(&urls)
}
