package goprocess

import (
	"sync"
)

// closedCh is an alread-closed channel. used to return
// in cases where we already know a channel is closed.
var closedCh chan struct{}

func init() {
	closedCh = make(chan struct{})
	close(closedCh)
}

// a processLink is an internal bookkeeping datastructure.
// it's used to form a relationship between two processes.
// It is mostly for keeping memory usage down (letting
// children close and be garbage-collected).
type processLink struct {
	// guards all fields.
	// DO NOT HOLD while holding process locks.
	// it may be slow, and could deadlock if not careful.
	sync.Mutex
	parent Process
	child  Process
}

func newProcessLink(p, c Process) *processLink {
	return &processLink{
		parent: p,
		child:  c,
	}
}

// Closing returns whether the child is closing
func (pl *processLink) ChildClosing() <-chan struct{} {
	// grab a hold of it, and unlock, as .Closing may block.
	pl.Lock()
	child := pl.child
	pl.Unlock()

	if child == nil { // already closed? memory optimization.
		return closedCh
	}
	return child.Closing()
}

func (pl *processLink) ChildClosed() <-chan struct{} {
	// grab a hold of it, and unlock, as .Closed may block.
	pl.Lock()
	child := pl.child
	pl.Unlock()

	if child == nil { // already closed? memory optimization.
		return closedCh
	}
	return child.Closed()
}

func (pl *processLink) ChildClose() {
	// grab a hold of it, and unlock, as .Closed may block.
	pl.Lock()
	child := pl.child
	pl.Unlock()

	if child != nil { // already closed? memory optimization.
		child.Close()
	}
}

func (pl *processLink) ClearChild() {
	pl.Lock()
	pl.child = nil
	pl.Unlock()
}

func (pl *processLink) ParentClear() {
	pl.Lock()
	pl.parent = nil
	pl.Unlock()
}

func (pl *processLink) Child() Process {
	pl.Lock()
	defer pl.Unlock()
	return pl.child
}

func (pl *processLink) Parent() Process {
	pl.Lock()
	defer pl.Unlock()
	return pl.parent
}

func (pl *processLink) AddToChild() {
	cp := pl.Child()

	// is it a *process ? if not... panic.
	c, ok := cp.(*process)
	if !ok {
		panic("goprocess does not yet support other process impls.")
	}

	// first, is it Closed?
	c.Lock()
	select {
	case <-c.Closed():
		c.Unlock()

		// already closed. must not add.
		// we must clear it, though. do so without the lock.
		pl.ClearChild()
		return

	default:
		// put the process link into q's waiters
		c.waiters = append(c.waiters, pl)
		c.Unlock()
	}
}
