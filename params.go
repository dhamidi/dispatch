package dispatch

// Params holds route parameters after extraction, default application, and
// normalization. Keys are case-sensitive. Values are always strings.
//
// Callers SHOULD treat a Params value returned inside a [Match] as read-only.
// Use [Params.Clone] to obtain a mutable copy.
type Params map[string]string

// Get returns the value associated with key, or an empty string if the key
// does not exist. Use [Params.Lookup] to distinguish between a missing key
// and a key whose value is the empty string.
func (p Params) Get(key string) string {
	return p[key]
}

// Lookup returns the value associated with key and whether the key was present.
func (p Params) Lookup(key string) (string, bool) {
	v, ok := p[key]
	return v, ok
}

// Clone returns a new Params containing all entries from p. Mutations to the
// returned map do not affect p.
func (p Params) Clone() Params {
	if p == nil {
		return nil
	}
	c := make(Params, len(p))
	for k, v := range p {
		c[k] = v
	}
	return c
}
