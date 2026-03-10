package dispatch

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ErrParamNotFound is returned when a named route parameter is not present
// in the request's match context.
var ErrParamNotFound = errors.New("dispatch: parameter not found")

// ParamError is returned when a parameter cannot be parsed to the requested type.
type ParamError struct {
	Name string // parameter name
	Raw  string // raw string value
	Err  error  // underlying parse error
}

func (e *ParamError) Error() string {
	return fmt.Sprintf("dispatch: parameter %q (value %q): %v", e.Name, e.Raw, e.Err)
}

func (e *ParamError) Unwrap() error { return e.Err }

// ParamValue converts a raw string parameter to a typed value.
// This mirrors flag.Value but is focused on parsing (not formatting).
type ParamValue interface {
	// Set parses the raw string value. Returns an error if parsing fails.
	Set(raw string) error
}

// paramRaw extracts the raw string value from the request's match context.
func paramRaw(r *http.Request, name string) (string, bool) {
	m, ok := MatchFromContext(r.Context())
	if !ok {
		return "", false
	}
	return m.Params.Lookup(name)
}

// ParamString returns the raw string value of the named route parameter.
// Returns ("", false) if the parameter is not present.
func ParamString(r *http.Request, name string) (string, bool) {
	return paramRaw(r, name)
}

// ParamInt returns the named route parameter parsed as an int.
// Returns (0, error) if missing or not a valid integer.
func ParamInt(r *http.Request, name string) (int, error) {
	raw, ok := paramRaw(r, name)
	if !ok {
		return 0, ErrParamNotFound
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &ParamError{Name: name, Raw: raw, Err: err}
	}
	return v, nil
}

// ParamInt64 returns the named route parameter parsed as an int64.
// Returns (0, error) if missing or not a valid int64.
func ParamInt64(r *http.Request, name string) (int64, error) {
	raw, ok := paramRaw(r, name)
	if !ok {
		return 0, ErrParamNotFound
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, &ParamError{Name: name, Raw: raw, Err: err}
	}
	return v, nil
}

// ParamFloat64 returns the named route parameter parsed as a float64.
// Returns (0, error) if missing or not a valid float64.
func ParamFloat64(r *http.Request, name string) (float64, error) {
	raw, ok := paramRaw(r, name)
	if !ok {
		return 0, ErrParamNotFound
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, &ParamError{Name: name, Raw: raw, Err: err}
	}
	return v, nil
}

// ParamBool returns the named route parameter parsed as a boolean.
// Accepts "true", "false", "1", "0", "yes", "no" (case-insensitive).
func ParamBool(r *http.Request, name string) (bool, error) {
	raw, ok := paramRaw(r, name)
	if !ok {
		return false, ErrParamNotFound
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, &ParamError{
			Name: name,
			Raw:  raw,
			Err:  fmt.Errorf("unrecognized boolean value %q", raw),
		}
	}
}

// MustParamInt returns the named parameter as int, panicking if extraction
// fails. Use only when a matching constraint guarantees validity.
func MustParamInt(r *http.Request, name string) int {
	v, err := ParamInt(r, name)
	if err != nil {
		panic(err)
	}
	return v
}

// MustParamInt64 returns the named parameter as int64, panicking if extraction
// fails. Use only when a matching constraint guarantees validity.
func MustParamInt64(r *http.Request, name string) int64 {
	v, err := ParamInt64(r, name)
	if err != nil {
		panic(err)
	}
	return v
}

// ParamAs extracts the named parameter and parses it into the given
// ParamValue. Returns an error if the parameter is missing or Set fails.
func ParamAs(r *http.Request, name string, dest ParamValue) error {
	raw, ok := paramRaw(r, name)
	if !ok {
		return ErrParamNotFound
	}
	if err := dest.Set(raw); err != nil {
		return &ParamError{Name: name, Raw: raw, Err: err}
	}
	return nil
}
