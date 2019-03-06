package cid

// Set is a implementation of a set of Cids, that is, a structure
// to which holds a single copy of every Cids that is added to it.
type Set struct {
	set map[Cid]struct{}
}

// NewSet initializes and returns a new Set.
func NewSet() *Set {
	return &Set{set: make(map[Cid]struct{})}
}

// Add puts a Cid in the Set.
func (s *Set) Add(c Cid) {
	s.set[c] = struct{}{}
}

// Has returns if the Set contains a given Cid.
func (s *Set) Has(c Cid) bool {
	_, ok := s.set[c]
	return ok
}

// Remove deletes a Cid from the Set.
func (s *Set) Remove(c Cid) {
	delete(s.set, c)
}

// Len returns how many elements the Set has.
func (s *Set) Len() int {
	return len(s.set)
}

// Keys returns the Cids in the set.
func (s *Set) Keys() []Cid {
	out := make([]Cid, 0, len(s.set))
	for k := range s.set {
		out = append(out, k)
	}
	return out
}

// Visit adds a Cid to the set only if it is
// not in it already.
func (s *Set) Visit(c Cid) bool {
	if !s.Has(c) {
		s.Add(c)
		return true
	}

	return false
}

// ForEach allows to run a custom function on each
// Cid in the set.
func (s *Set) ForEach(f func(c Cid) error) error {
	for c := range s.set {
		err := f(c)
		if err != nil {
			return err
		}
	}
	return nil
}
