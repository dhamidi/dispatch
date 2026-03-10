// Package dispatch provides semantic HTTP routing for Go.
//
// It is inspired by Rails ActionDispatch while remaining idiomatic to Go
// and compatible with the standard net/http package.
//
// Routing is built on URI templates (github.com/dhamidi/uritemplate) as the
// single source of truth for both inbound matching and outbound URL generation.
package dispatch

