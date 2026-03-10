package dispatch

import "errors"

// Sentinel errors returned by registration and URL generation operations.
var (
	// ErrDuplicateRoute is returned when a route name is already registered.
	ErrDuplicateRoute = errors.New("dispatch: duplicate route name")

	// ErrEmptyRouteName is returned when a route has an empty name.
	ErrEmptyRouteName = errors.New("dispatch: empty route name")

	// ErrNilTemplate is returned when a route has a nil template.
	ErrNilTemplate = errors.New("dispatch: nil template")

	// ErrNilHandler is returned when a dispatchable route has a nil handler.
	ErrNilHandler = errors.New("dispatch: nil handler")

	// ErrUnknownRoute is returned when URL generation references an unknown name.
	ErrUnknownRoute = errors.New("dispatch: unknown route name")

	// ErrMissingParam is returned when a required template variable is absent
	// during URL generation.
	ErrMissingParam = errors.New("dispatch: missing required parameter")

	// ErrMethodNotAllowed is returned / emitted when a URL structurally matches
	// one or more routes but none allow the request method.
	ErrMethodNotAllowed = errors.New("dispatch: method not allowed")

	// ErrNotFound is returned / emitted when no route matches the request.
	ErrNotFound = errors.New("dispatch: not found")
)
