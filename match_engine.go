package dispatch

import (
	"net/http"
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
		canonical, err := computeCanonicalURL(best.route, best.params)
		if err == nil && canonical != nil {
			m.CanonicalURL = canonical
			m.IsCanonical = isCanonicalURL(&reqURL, canonical)

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

	for _, reg := range r.routes {
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
			score := reg.score // start from precomputed structural hints
			// Compute dynamic query matches at match time
			if rc.URL != nil {
				vars := templateVarNames(route.Template)
				for key := range rc.URL.Query() {
					if _, ok := vars[key]; ok {
						score.QueryMatches++
					}
				}
			}
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

// methodBit maps an HTTP method string to its [MethodSet] constant,
// returning 0 for unknown methods.
func methodBit(method string) MethodSet {
	return methodFromString(method)
}

// --- utility functions ------------------------------------------------------

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

