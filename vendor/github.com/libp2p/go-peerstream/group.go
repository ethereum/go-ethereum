package peerstream

import (
	"errors"
	"sync"
	"unsafe"
)

// ErrGroupNotFound signals no such group exists
var ErrGroupNotFound = errors.New("group not found")

// Group is an object used to associate a group of
// Streams, Connections, and Listeners. It can be anything,
// it is meant to work like a KeyType in maps
type Group interface{}

// Groupable is an interface for a set of objects that can
// be assigned groups: Streams, Connections, and Listeners.
// Objects inherit groups (e.g. a Stream inherits the groups
// of its parent Connection, and in turn that of its Listener).
type Groupable interface {
	// Groups returns the groups this object belongs to
	Groups() []Group

	// InGroup returns whether this object belongs to a Group
	InGroup(g Group) bool

	// AddGroup adds this object to a group
	AddGroup(g Group)
}

// groupSet is a struct designed to be embedded and
// give things group memebership
type groupSet struct {
	m map[Group]struct{}
	sync.RWMutex
}

func (gs *groupSet) Add(g Group) {
	gs.Lock()
	defer gs.Unlock()
	gs.m[g] = struct{}{}
}

func (gs *groupSet) Remove(g Group) {
	gs.Lock()
	defer gs.Unlock()
	delete(gs.m, g)
}

func (gs *groupSet) Has(g Group) bool {
	gs.RLock()
	defer gs.RUnlock()
	_, ok := gs.m[g]
	return ok
}

func (gs *groupSet) Groups() []Group {
	gs.RLock()
	defer gs.RUnlock()

	out := make([]Group, 0, len(gs.m))
	for k := range gs.m {
		out = append(out, k)
	}
	return out
}

// AddSet adds all elements in another set.
func (gs *groupSet) AddSet(gs2 *groupSet) {
	// acquire locks in order
	p1 := uintptr(unsafe.Pointer(gs))
	p2 := uintptr(unsafe.Pointer(gs2))
	switch {
	case p1 < p2:
		gs.Lock()
		gs2.RLock()
		defer gs.Unlock()
		defer gs2.RUnlock()
	case p1 > p2:
		gs2.Lock()
		gs.Lock()
		defer gs2.Unlock()
		defer gs.Unlock()
	default:
		return // they're the same!
	}

	for g := range gs2.m {
		gs.m[g] = struct{}{}
	}
}
