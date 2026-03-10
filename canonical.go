package dispatch

import (
	"net/url"
	"sort"
	"strings"
)

// computeCanonicalURL expands route.Template with params and returns the
// resulting *url.URL. Returns nil and an error if expansion fails.
func computeCanonicalURL(route *Route, params Params) (*url.URL, error) {
	expanded, err := expandTemplate(route.Template, params)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(expanded)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// isCanonicalURL reports whether reqURL is equivalent to canonicalURL for
// routing purposes. Comparison considers path and sorted query string only;
// scheme and host are ignored.
func isCanonicalURL(reqURL, canonicalURL *url.URL) bool {
	if reqURL.Path != canonicalURL.Path {
		return false
	}
	return normalizeQuery(reqURL.RawQuery) == normalizeQuery(canonicalURL.RawQuery)
}

// normalizeQuery returns a canonical query string with keys sorted
// alphabetically. Values within each key are preserved in their original order.
func normalizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return raw // cannot normalize malformed query
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range values[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}
