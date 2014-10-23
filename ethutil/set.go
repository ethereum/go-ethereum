package ethutil

type Settable interface {
	AsSet() UniqueSet
}

type Stringable interface {
	String() string
}

type UniqueSet map[string]struct{}

func NewSet(v ...Stringable) UniqueSet {
	set := make(UniqueSet)
	for _, val := range v {
		set.Insert(val)
	}

	return set
}

func (self UniqueSet) Insert(k Stringable) UniqueSet {
	self[k.String()] = struct{}{}

	return self
}

func (self UniqueSet) Include(k Stringable) bool {
	_, ok := self[k.String()]

	return ok
}

func Set(s Settable) UniqueSet {
	return s.AsSet()
}
