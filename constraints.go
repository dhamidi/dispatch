package dispatch

import (
	"regexp"
	"strconv"
	"strings"
)

// Exact returns a [Constraint] that accepts a candidate only when the
// parameter named key has exactly the given value.
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Exact("format", "json"))
func Exact(key, value string) Constraint {
	return ConstraintFunc(func(_ *RequestContext, p Params) bool {
		return p[key] == value
	})
}

// OneOf returns a [Constraint] that accepts a candidate only when the
// parameter named key matches one of the provided values.
//
// Example:
//
//	dispatch.WithConstraint(dispatch.OneOf("format", "json", "xml"))
func OneOf(key string, values ...string) Constraint {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return ConstraintFunc(func(_ *RequestContext, p Params) bool {
		_, ok := set[p[key]]
		return ok
	})
}

// Regexp returns a [Constraint] that accepts a candidate only when the
// parameter named key fully matches the compiled regular expression re.
//
// The regular expression is anchored automatically: the parameter value must
// match the entire expression, not just a substring. Use [regexp.MustCompile]
// to build the expression.
//
// Only Go standard library regexp facilities are used; no external dependency
// is introduced.
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Regexp("slug", regexp.MustCompile(`^[a-z0-9-]+$`)))
func Regexp(key string, re *regexp.Regexp) Constraint {
	return ConstraintFunc(func(_ *RequestContext, p Params) bool {
		return re.MatchString(p[key])
	})
}

// Int returns a [Constraint] that accepts a candidate only when the parameter
// named key can be parsed as a base-10 integer (using [strconv.Atoi]).
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Int("id"))
func Int(key string) Constraint {
	return ConstraintFunc(func(_ *RequestContext, p Params) bool {
		_, err := strconv.Atoi(p[key])
		return err == nil
	})
}

// Host returns a [Constraint] that accepts a candidate only when the request
// host matches the given host string exactly (case-insensitive comparison
// against [RequestContext.Host]).
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Host("api.example.com"))
func Host(host string) Constraint {
	lower := strings.ToLower(host)
	return ConstraintFunc(func(rc *RequestContext, _ Params) bool {
		return strings.ToLower(rc.Host) == lower
	})
}

// Methods returns a [Constraint] that accepts a candidate only when the
// request method is included in the given [MethodSet].
//
// This constraint is useful when a route is registered with multiple methods
// but a specific handler variant should apply only to a subset.
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Methods(dispatch.GET | dispatch.HEAD))
func Methods(ms MethodSet) Constraint {
	return ConstraintFunc(func(rc *RequestContext, _ Params) bool {
		bit := methodBit(rc.Method)
		return ms.Has(bit)
	})
}

// Custom returns a [Constraint] from an arbitrary predicate function. It is a
// thin convenience wrapper around [ConstraintFunc].
//
// Example:
//
//	dispatch.WithConstraint(dispatch.Custom(func(rc *dispatch.RequestContext, p dispatch.Params) bool {
//	    return p["version"] != ""
//	}))
func Custom(fn func(*RequestContext, Params) bool) Constraint {
	return ConstraintFunc(fn)
}
