package dispatch

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/dhamidi/uritemplate"
)

// Match resolves req against the registered routes and returns the best
// [Match], or an error.
//
// Errors:
//   - [ErrNotFound]: no route matched the request URL.
//   - [ErrMethodNotAllowed]: at least one route matched structurally but none
//     allowed the request method.
//
// On success the returned Match contains the selected route, resolved Params,
// canonical URL information (if applicable), and a RedirectNeeded flag.
func (r *Router) Match(req *http.Request) (*Match, error) {
	// Phase 1 — Normalize request method and URL
	method := strings.ToUpper(req.Method)
	reqURL := *req.URL // clone to avoid mutating the original
	host := req.Host
	if h, _, err := strings.Cut(host, ":"); err {
		host = h
	}

	rc := &RequestContext{
		Request: req,
		URL:     &reqURL,
		Method:  method,
		Host:    host,
	}

	// Phase 2–8 — Enumerate, filter, score candidates
	candidates, methodMismatch := r.filterCandidates(rc)

	// Phase 9 — Select best candidate
	if len(candidates) == 0 {
		if methodMismatch {
			return nil, ErrMethodNotAllowed
		}
		return nil, ErrNotFound
	}

	best := r.selectBest(candidates)
	if best == nil {
		return nil, ErrNotFound
	}

	m := &Match{
		Route:  best.route,
		Name:   best.route.Name,
		Params: best.params,
		Method: method,
		score:  best.score,
	}

	// Phase 10 — Compute canonical URL
	policy := best.route.CanonicalPolicy
	if policy == CanonicalIgnore {
		policy = r.config.defaultCanonicalPolicy
	}
	if policy != CanonicalIgnore {
		canonical, err := r.computeCanonical(best.route, best.params)
		if err == nil && canonical != nil {
			m.CanonicalURL = canonical
			m.IsCanonical = isCanonicalMatch(&reqURL, canonical)

			// Phase 11 — Decide dispatch, redirect, or rejection
			if !m.IsCanonical {
				switch policy {
				case CanonicalRedirect:
					m.RedirectNeeded = true
				case CanonicalReject:
					return nil, ErrNotFound
				}
			}
		}
	}

	return m, nil
}

// candidate is the internal representation of a route during match resolution.
type candidate struct {
	route  *Route
	params Params
	score  candidateScore
}

// filterCandidates returns all routes that pass method + template + query +
// constraint filtering. It also detects method-not-allowed situations.
func (r *Router) filterCandidates(rc *RequestContext) (matched []*candidate, methodMismatch bool) {
	reqMethod := methodBit(rc.Method)

	// Build the request URI for template matching
	matchURI := rc.URL.RequestURI()

	for i, reg := range r.routes {
		route := &reg.Route

		// Phase 4 — URI template reverse match
		vals, ok := route.Template.Match(matchURI)
		if !ok {
			// Also try matching with just the URL for templates without query expressions
			vals, ok = route.Template.FromURL(rc.URL)
			if !ok {
				continue
			}
		}

		// Phase 3 — Filter by method compatibility
		methodOK := route.Methods.Has(reqMethod)
		if !methodOK && r.config.implicitHEADFromGET && reqMethod == HEAD {
			methodOK = route.Methods.Has(GET)
		}
		if !methodOK {
			methodMismatch = true
			continue
		}

		// Phase 5 — Apply defaults
		params := valuesToParams(vals)
		if route.Defaults != nil {
			for k, v := range route.Defaults {
				if _, exists := params[k]; !exists {
					params[k] = v
				}
			}
		}

		// Phase 6 — Enforce query mode
		qm := route.QueryMode
		if qm == QueryLoose {
			qm = r.config.defaultQueryMode
		}
		if qm == QueryStrict {
			declaredVars := templateVarNames(route.Template)
			for key := range rc.URL.Query() {
				if _, declared := declaredVars[key]; !declared {
					goto nextRoute
				}
			}
		}

		// Phase 7 — Evaluate constraints
		{
			constraintOK := true
			for _, c := range route.Constraints {
				if !c.Check(rc, params) {
					constraintOK = false
					break
				}
			}
			if !constraintOK {
				continue
			}
		}

		// Phase 8 — Score candidate
		{
			score := computeScore(route, params, rc.URL, i)
			matched = append(matched, &candidate{
				route:  route,
				params: params,
				score:  score,
			})
		}
		continue

	nextRoute:
	}
	return matched, methodMismatch
}

// selectBest picks the highest-scoring candidate deterministically.
func (r *Router) selectBest(candidates []*candidate) *candidate {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score.beats(best.score) {
			best = c
		}
	}
	return best
}

// computeCanonical expands the matched route template with final params and
// returns the canonical *url.URL.
func (r *Router) computeCanonical(route *Route, params Params) (*url.URL, error) {
	vals := paramsToValues(params)
	expanded, err := route.Template.Expand(vals)
	if err != nil {
		return nil, err
	}
	return url.Parse(expanded)
}

// methodBit maps an HTTP method string to its [MethodSet] constant,
// returning 0 for unknown methods.
func methodBit(method string) MethodSet {
	return methodFromString(method)
}

// isCanonicalMatch reports whether the request URL and the canonical URL are
// equivalent. Paths are compared case-sensitively. Query parameters are
// compared order-independently.
func isCanonicalMatch(reqURL, canonical *url.URL) bool {
	if reqURL == nil || canonical == nil {
		return true
	}
	if reqURL.Path != canonical.Path {
		return false
	}
	// Compare query parameters (order-independent)
	reqQuery := reqURL.Query()
	canonQuery := canonical.Query()
	if len(reqQuery) != len(canonQuery) {
		return false
	}
	for k, rv := range reqQuery {
		cv, ok := canonQuery[k]
		if !ok || len(rv) != len(cv) {
			return false
		}
		for i := range rv {
			if rv[i] != cv[i] {
				return false
			}
		}
	}
	return true
}

// --- utility functions ------------------------------------------------------

// paramsToValues converts [Params] to [uritemplate.Values].
func paramsToValues(p Params) uritemplate.Values {
	vals := make(uritemplate.Values, len(p))
	for k, v := range p {
		vals[k] = uritemplate.String(v)
	}
	return vals
}

// valuesToParams converts [uritemplate.Values] to [Params] by expanding each
// value through a single-variable template.
func valuesToParams(vals uritemplate.Values) Params {
	if vals == nil {
		return make(Params)
	}
	p := make(Params, len(vals))
	for k, v := range vals {
		t := uritemplate.MustParse("{v}")
		expanded, err := t.Expand(uritemplate.Values{"v": v})
		if err == nil {
			p[k] = expanded
		}
	}
	return p
}

// mergeParams merges provided params with defaults. Provided params take
// precedence.
func mergeParams(provided, defaults Params) Params {
	result := make(Params)
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range provided {
		result[k] = v
	}
	return result
}

// templateVarNames extracts variable names declared in a [uritemplate.Template]
// by parsing the raw template string.
func templateVarNames(t *uritemplate.Template) map[string]struct{} {
	raw := t.String()
	vars := make(map[string]struct{})
	i := 0
	for i < len(raw) {
		if raw[i] == '{' {
			end := strings.IndexByte(raw[i:], '}')
			if end < 0 {
				break
			}
			body := raw[i+1 : i+end]
			// Skip operator character if present
			if len(body) > 0 {
				first := body[0]
				if first == '+' || first == '#' || first == '.' || first == '/' || first == ';' || first == '?' || first == '&' {
					body = body[1:]
				}
			}
			// Split by comma for multiple variables
			parts := strings.Split(body, ",")
			for _, p := range parts {
				name := strings.TrimRight(p, "*")
				if colonIdx := strings.IndexByte(name, ':'); colonIdx >= 0 {
					name = name[:colonIdx]
				}
				name = strings.TrimSpace(name)
				if name != "" {
					vars[name] = struct{}{}
				}
			}
			i += end + 1
		} else {
			i++
		}
	}
	return vars
}

// computeScore computes the [candidateScore] for a route given its match
// parameters and the request URL.
func computeScore(route *Route, params Params, reqURL *url.URL, registrationIdx int) candidateScore {
	raw := route.Template.String()

	// Count literal characters (non-expression, non-separator)
	litChars := 0
	inExpr := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '{' {
			inExpr = true
		} else if raw[i] == '}' {
			inExpr = false
		} else if !inExpr {
			litChars++
		}
	}

	// Count query matches
	queryMatches := 0
	vars := templateVarNames(route.Template)
	if reqURL != nil {
		for key := range reqURL.Query() {
			if _, ok := vars[key]; ok {
				queryMatches++
			}
		}
	}

	constrainedVars := len(route.Constraints)
	broadVars := len(vars) - constrainedVars
	if broadVars < 0 {
		broadVars = 0
	}

	return candidateScore{
		LiteralSegments: litChars,
		ConstrainedVars: constrainedVars,
		BroadVars:       broadVars,
		QueryMatches:    queryMatches,
		Priority:        route.Priority,
		Registration:    registrationIdx,
	}
}
