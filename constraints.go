package dispatch

import (
	"regexp"
	"strconv"
)

// Exact returns a Constraint that passes only when params[key] == value.
func Exact(key, value string) Constraint {
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		return p.Get(key) == value
	})
}

// OneOf returns a Constraint that passes when params[key] is one of values.
func OneOf(key string, values ...string) Constraint {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		_, ok := set[p.Get(key)]
		return ok
	})
}

// Regexp returns a Constraint that passes when params[key] fully matches re.
// Uses Go standard library regexp (§16.1); no external dependencies.
func Regexp(key string, re *regexp.Regexp) Constraint {
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		v := p.Get(key)
		loc := re.FindStringIndex(v)
		if loc == nil {
			return false
		}
		return loc[0] == 0 && loc[1] == len(v)
	})
}

// Int returns a Constraint that passes when params[key] is a valid integer
// (parseable by strconv.Atoi).
func Int(key string) Constraint {
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		_, err := strconv.Atoi(p.Get(key))
		return err == nil
	})
}

// Host returns a Constraint that passes when rc.Host equals host.
func Host(host string) Constraint {
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		return rc.Host == host
	})
}

// Methods returns a Constraint that passes when rc.Method is in ms.
func Methods(ms MethodSet) Constraint {
	return ConstraintFunc(func(rc *RequestContext, p Params) bool {
		return ms.contains(methodFromString(rc.Method))
	})
}

// Custom wraps an arbitrary function as a Constraint.
func Custom(fn func(*RequestContext, Params) bool) Constraint {
	return ConstraintFunc(fn)
}
