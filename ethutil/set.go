package ethutil

type Settable interface {
	AsSet() UniqueSet
}

type UniqueSet map[interface{}]struct{}

func NewSet(v ...interface{}) UniqueSet {
	set := make(UniqueSet)
	for _, val := range v {
		set.Insert(val)
	}

	return set
}

func (self UniqueSet) Insert(k interface{}) UniqueSet {
	self[k] = struct{}{}

	return self
}

func (self UniqueSet) Include(k interface{}) bool {
	_, ok := self[k]

	return ok
}

func Set(s Settable) UniqueSet {
	return s.AsSet()
}
