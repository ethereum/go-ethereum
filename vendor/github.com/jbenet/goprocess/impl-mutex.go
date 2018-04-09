package goprocess

import (
	"sync"
)

// process implements Process
type process struct {
	children map[*processLink]struct{} // process to close with us
	waitfors map[*processLink]struct{} // process to only wait for
	waiters  []*processLink            // processes that wait for us. for gc.

	teardown TeardownFunc  // called to run the teardown logic.
	waiting  chan struct{} // closed when CloseAfterChildrenClosed is called.
	closing  chan struct{} // closed once close starts.
	closed   chan struct{} // closed once close is done.
	closeErr error         // error to return to clients of Close()

	sync.Mutex
}

// newProcess constructs and returns a Process.
// It will call tf TeardownFunc exactly once:
//  **after** all children have fully Closed,
//  **after** entering <-Closing(), and
//  **before** <-Closed().
func newProcess(tf TeardownFunc) *process {
	return &process{
		teardown: tf,
		closed:   make(chan struct{}),
		closing:  make(chan struct{}),
		waitfors: make(map[*processLink]struct{}),
		children: make(map[*processLink]struct{}),
	}
}

func (p *process) WaitFor(q Process) {
	if q == nil {
		panic("waiting for nil process")
	}

	p.Lock()

	select {
	case <-p.Closed():
		panic("Process cannot wait after being closed")
	default:
	}

	pl := newProcessLink(p, q)
	p.waitfors[pl] = struct{}{}
	p.Unlock()
	go pl.AddToChild()
}

func (p *process) AddChildNoWait(child Process) {
	if child == nil {
		panic("adding nil child process")
	}

	p.Lock()

	select {
	case <-p.Closed():
		panic("Process cannot add children after being closed")
	case <-p.Closing():
		go child.Close()
	default:
	}

	pl := newProcessLink(p, child)
	p.children[pl] = struct{}{}
	p.Unlock()
	go pl.AddToChild()
}

func (p *process) AddChild(child Process) {
	if child == nil {
		panic("adding nil child process")
	}

	p.Lock()

	select {
	case <-p.Closed():
		panic("Process cannot add children after being closed")
	case <-p.Closing():
		go child.Close()
	default:
	}

	pl := newProcessLink(p, child)
	if p.waitfors != nil { // if p.waitfors hasn't been set nil
		p.waitfors[pl] = struct{}{}
	}
	if p.children != nil { // if p.children hasn't been set nil
		p.children[pl] = struct{}{}
	}
	p.Unlock()
	go pl.AddToChild()
}

func (p *process) Go(f ProcessFunc) Process {
	child := newProcess(nil)
	waitFor := newProcess(nil)
	child.WaitFor(waitFor) // prevent child from closing

	// add child last, to prevent a closing parent from
	// closing all of them prematurely, before running the func.
	p.AddChild(child)
	go func() {
		f(child)
		waitFor.Close()            // allow child to close.
		child.CloseAfterChildren() // close to tear down.
	}()
	return child
}

// SetTeardown to assign a teardown function
func (p *process) SetTeardown(tf TeardownFunc) {
	if tf == nil {
		panic("cannot set nil TeardownFunc")
	}

	p.Lock()
	if p.teardown != nil {
		panic("cannot SetTeardown twice")
	}

	p.teardown = tf
	select {
	case <-p.Closed():
		p.closeErr = tf()
	default:
	}
	p.Unlock()
}

// Close is the external close function.
// it's a wrapper around internalClose that waits on Closed()
func (p *process) Close() error {
	p.Lock()

	// if already closing, or closed, get out. (but wait!)
	select {
	case <-p.Closing():
		p.Unlock()
		<-p.Closed()
		return p.closeErr
	default:
	}

	p.doClose()
	p.Unlock()
	return p.closeErr
}

func (p *process) Closing() <-chan struct{} {
	return p.closing
}

func (p *process) Closed() <-chan struct{} {
	return p.closed
}

func (p *process) Err() error {
	<-p.Closed()
	return p.closeErr
}

// the _actual_ close process.
func (p *process) doClose() {
	// this function is only be called once (protected by p.Lock()).
	// and it will panic (on closing channels) otherwise.

	close(p.closing) // signal that we're shutting down (Closing)

	for len(p.children) > 0 || len(p.waitfors) > 0 {
		for plc, _ := range p.children {
			child := plc.Child()
			if child != nil { // check because child may already have been removed.
				go child.Close() // force all children to shut down
			}
			plc.ParentClear()
		}
		p.children = nil // clear them. release memory.

		// we must be careful not to iterate over waitfors directly, as it may
		// change under our feet.
		wf := p.waitfors
		p.waitfors = nil // clear them. release memory.
		for w, _ := range wf {
			// Here, we wait UNLOCKED, so that waitfors who are in the middle of
			// adding a child to us can finish. we will immediately close the child.
			p.Unlock()
			<-w.ChildClosed() // wait till all waitfors are fully closed (before teardown)
			p.Lock()
			w.ParentClear()
		}
	}

	if p.teardown != nil {
		p.closeErr = p.teardown() // actually run the close logic (ok safe to teardown)
	}
	close(p.closed) // signal that we're shut down (Closed)

	// go remove all the parents from the process links. optimization.
	go func(waiters []*processLink) {
		for _, pl := range waiters {
			pl.ClearChild()
			pr, ok := pl.Parent().(*process)
			if !ok {
				// parent has already been called to close
				continue
			}
			pr.Lock()
			delete(pr.waitfors, pl)
			delete(pr.children, pl)
			pr.Unlock()
		}
	}(p.waiters) // pass in so
	p.waiters = nil // clear them. release memory.
}

// We will only wait on the children we have now.
// We will not wait on children added subsequently.
// this may change in the future.
func (p *process) CloseAfterChildren() error {
	p.Lock()
	select {
	case <-p.Closed():
		p.Unlock()
		return p.Close() // get error. safe, after p.Closed()
	case <-p.waiting: // already called it.
		p.Unlock()
		<-p.Closed()
		return p.Close() // get error. safe, after p.Closed()
	default:
	}
	p.Unlock()

	// here only from one goroutine.

	nextToWaitFor := func() Process {
		p.Lock()
		defer p.Unlock()
		for e, _ := range p.waitfors {
			c := e.Child()
			if c == nil {
				continue
			}

			select {
			case <-c.Closed():
			default:
				return c
			}
		}
		return nil
	}

	// wait for all processes we're waiting for are closed.
	// the semantics here are simple: we will _only_ close
	// if there are no processes currently waiting for.
	for next := nextToWaitFor(); next != nil; next = nextToWaitFor() {
		<-next.Closed()
	}

	// YAY! we're done. close
	return p.Close()
}
