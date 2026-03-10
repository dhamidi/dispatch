package dispatch

// Constraint refines a candidate route after URI template extraction. It
// receives the full [RequestContext] and the resolved [Params] (with defaults
// already applied) and returns true if the candidate satisfies the constraint.
//
// Constraints MUST be side-effect free and MUST NOT mutate the supplied Params.
// Multiple constraints registered on a single route are evaluated in
// registration order; the first failing constraint rejects the candidate.
type Constraint interface {
	Check(*RequestContext, Params) bool
}

// ConstraintFunc is a function adapter that implements [Constraint].
// It allows plain functions to satisfy the Constraint interface without
// defining a named type.
//
// Example:
//
//	var adminOnly dispatch.ConstraintFunc = func(rc *dispatch.RequestContext, p dispatch.Params) bool {
//	    return rc.Request.Header.Get("X-Admin") == "true"
//	}
type ConstraintFunc func(*RequestContext, Params) bool

// Check calls f(rc, p) and returns its result, satisfying [Constraint].
func (f ConstraintFunc) Check(rc *RequestContext, p Params) bool {
	return f(rc, p)
}

// Compile-time check that ConstraintFunc satisfies Constraint.
var _ Constraint = ConstraintFunc(nil)
