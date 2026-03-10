package dispatch

import (
	"net/http"
	"net/url"
)

// RequestContext holds the normalized attributes of an incoming HTTP request
// that are relevant during route matching and constraint evaluation.
//
// RequestContext is constructed by the router for each request and passed to
// every [Constraint.Check] call. Constraints MUST NOT mutate it.
type RequestContext struct {
	// Request is the original *http.Request. It is always non-nil during
	// route matching.
	Request *http.Request

	// URL is the normalized request URL. The router may have cleaned or
	// decoded the URL before matching; prefer this field over Request.URL
	// inside constraints.
	URL *url.URL

	// Method is the normalized (upper-cased) HTTP method string, for example
	// "GET" or "POST".
	Method string

	// Host is the request host extracted from the request, without port
	// unless the port is non-standard.
	Host string
}
