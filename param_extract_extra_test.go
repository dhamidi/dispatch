package dispatch

import (
	"errors"
	"testing"
)

func TestParamError_ErrorMessage(t *testing.T) {
	e := &ParamError{Name: "id", Raw: "abc", Err: errors.New("not a number")}
	got := e.Error()
	if !containsSubstring(got, "id") || !containsSubstring(got, "abc") {
		t.Errorf("error message should contain name and raw value, got %q", got)
	}
}
