// Package goprocess introduces a Process abstraction that allows simple
// organization, and orchestration of work. It is much like a WaitGroup,
// and much like a context.Context, but also ensures safe **exactly-once**,
// and well-ordered teardown semantics.
package goprocess

import (
	"os"
	"os/signal"
)

// Process is the basic unit of work in goprocess. It defines a computation
// with a lifecycle:
// - running (before calling Close),
// - closing (after calling Close at least once),
// - closed (after Close returns, and all teardown has _completed_).
//
// More specifically, it fits this:
//
//   p := WithTeardown(tf) // new process is created, it is now running.
//   p.AddChild(q)         // can register children **before** Closed().
//   go p.Close()          // blocks until done running teardown func.
//   <-p.Closing()         // would now return true.
//   <-p.childrenDone()    // wait on all children to be done
//   p.teardown()          // runs the user's teardown function tf.
//   p.Close()             // now returns, with error teardown returned.
//   <-p.Closed()          // would now return true.
//
// Processes can be arranged in a process "tree", where children are
// automatically Closed if their parents are closed. (Note, it is actually
// a Process DAG, children may have multiple parents). A process may also
// optionally wait for another to fully Close before beginning to Close.
// This makes it easy to ensure order of operations and proper sequential
// teardown of resurces. For example:
//
//   p1 := goprocess.WithTeardown(func() error {
//     fmt.Println("closing 1")
//   })
//   p2 := goprocess.WithTeardown(func() error {
//     fmt.Println("closing 2")
//   })
//   p3 := goprocess.WithTeardown(func() error {
//     fmt.Println("closing 3")
//   })
//
//   p1.AddChild(p2)
//   p2.AddChild(p3)
//
//
//   go p1.Close()
//   go p2.Close()
//   go p3.Close()
//
//   // Output:
//   // closing 3
//   // closing 2
//   // closing 1
//
// Process is modelled after the UNIX processes group idea, and heavily
// informed by sync.WaitGroup and go.net/context.Context.
//
// In the function documentation of this interface, `p` always refers to
// the self Process.
type Process interface {

	// WaitFor makes p wait for q before exiting. Thus, p will _always_ close
	// _after_ q. Note well: a waiting cycle is deadlock.
	//
	// If q is already Closed, WaitFor calls p.Close()
	// If p is already Closing or Closed, WaitFor panics. This is the same thing
	// as calling Add(1) _after_ calling Done() on a wait group. Calling WaitFor
	// on an already-closed process is a programming error likely due to bad
	// synchronization
	WaitFor(q Process)

	// AddChildNoWait registers child as a "child" of Process. As in UNIX,
	// when parent is Closed, child is Closed -- child may Close beforehand.
	// This is the equivalent of calling:
	//
	//  go func(parent, child Process) {
	//    <-parent.Closing()
	//    child.Close()
	//  }(p, q)
	//
	// Note: the naming of functions is `AddChildNoWait` and `AddChild` (instead
	// of `AddChild` and `AddChildWaitFor`) because:
	// - it is the more common operation,
	// - explicitness is helpful in the less common case (no waiting), and
	// - usual "child" semantics imply parent Processes should wait for children.
	AddChildNoWait(q Process)

	// AddChild is the equivalent of calling:
	//  parent.AddChildNoWait(q)
	//  parent.WaitFor(q)
	AddChild(q Process)

	// Go is much like `go`, as it runs a function in a newly spawned goroutine.
	// The neat part of Process.Go is that the Process object you call it on will:
	//  * construct a child Process, and call AddChild(child) on it
	//  * spawn a goroutine, and call the given function
	//  * Close the child when the function exits.
	// This way, you can rest assured each goroutine you spawn has its very own
	// Process context, and that it will be closed when the function exits.
	// It is the function's responsibility to respect the Closing of its Process,
	// namely it should exit (return) when <-Closing() is ready. It is basically:
	//
	//   func (p Process) Go(f ProcessFunc) Process {
	//   	child := WithParent(p)
	//   	go func () {
	//   		f(child)
	//   		child.Close()
	//   	}()
	//   }
	//
	// It is useful to construct simple asynchronous workers, children of p.
	Go(f ProcessFunc) Process

	// SetTeardown sets the process's teardown to tf.
	SetTeardown(tf TeardownFunc)

	// Close ends the process. Close blocks until the process has completely
	// shut down, and any teardown has run _exactly once_. The returned error
	// is available indefinitely: calling Close twice returns the same error.
	// If the process has already been closed, Close returns immediately.
	Close() error

	// CloseAfterChildren calls Close _after_ its children have Closed
	// normally (i.e. it _does not_ attempt to close them).
	CloseAfterChildren() error

	// Closing is a signal to wait upon. The returned channel is closed
	// _after_ Close has been called at least once, but teardown may or may
	// not be done yet. The primary use case of Closing is for children who
	// need to know when a parent is shutting down, and therefore also shut
	// down.
	Closing() <-chan struct{}

	// Closed is a signal to wait upon. The returned channel is closed
	// _after_ Close has completed; teardown has finished. The primary use case
	// of Closed is waiting for a Process to Close without _causing_ the Close.
	Closed() <-chan struct{}

	// Err waits until the process is closed, and then returns any error that
	// occurred during shutdown.
	Err() error
}

// TeardownFunc is a function used to cleanup state at the end of the
// lifecycle of a Process.
type TeardownFunc func() error

// ProcessFunc is a function that takes a process. Its main use case is goprocess.Go,
// which spawns a ProcessFunc in its own goroutine, and returns a corresponding
// Process object.
type ProcessFunc func(proc Process)

var nilProcessFunc = func(Process) {}

// Go is much like `go`: it runs a function in a newly spawned goroutine. The neat
// part of Go is that it provides Process object to communicate between the
// function and the outside world. Thus, callers can easily WaitFor, or Close the
// function. It is the function's responsibility to respect the Closing of its Process,
// namely it should exit (return) when <-Closing() is ready. It is simply:
//
//   func Go(f ProcessFunc) Process {
//     p := WithParent(Background())
//     p.Go(f)
//     return p
//   }
//
// Note that a naive implementation of Go like the following would not work:
//
//   func Go(f ProcessFunc) Process {
//     return Background().Go(f)
//   }
//
// This is because having the process you
func Go(f ProcessFunc) Process {
	// return GoChild(Background(), f)

	// we use two processes, one for communication, and
	// one for ensuring we wait on the function (unclosable from the outside).
	p := newProcess(nil)
	waitFor := newProcess(nil)
	p.WaitFor(waitFor) // prevent p from closing
	go func() {
		f(p)
		waitFor.Close() // allow p to close.
		p.Close()       // ensure p closes.
	}()
	return p
}

// GoChild is like Go, but it registers the returned Process as a child of parent,
// **before** spawning the goroutine, which ensures proper synchronization with parent.
// It is somewhat like
//
//   func GoChild(parent Process, f ProcessFunc) Process {
//     p := WithParent(parent)
//     p.Go(f)
//     return p
//   }
//
// And it is similar to the classic WaitGroup use case:
//
//   func WaitGroupGo(wg sync.WaitGroup, child func()) {
//     wg.Add(1)
//     go func() {
//       child()
//       wg.Done()
//     }()
//   }
//
func GoChild(parent Process, f ProcessFunc) Process {
	p := WithParent(parent)
	p.Go(f)
	return p
}

// Spawn is an alias of `Go`. In many contexts, Spawn is a
// well-known Process launching word, which fits our use case.
var Spawn = Go

// SpawnChild is an alias of `GoChild`. In many contexts, Spawn is a
// well-known Process launching word, which fits our use case.
var SpawnChild = GoChild

// WithTeardown constructs and returns a Process with a TeardownFunc.
// TeardownFunc tf will be called **exactly-once** when Process is
// Closing, after all Children have fully closed, and before p is Closed.
// In fact, Process p will not be Closed until tf runs and exits.
// See lifecycle in Process doc.
func WithTeardown(tf TeardownFunc) Process {
	if tf == nil {
		panic("nil tf TeardownFunc")
	}
	return newProcess(tf)
}

// WithParent constructs and returns a Process with a given parent.
func WithParent(parent Process) Process {
	if parent == nil {
		panic("nil parent Process")
	}
	q := newProcess(nil)
	parent.AddChild(q)
	return q
}

// WithSignals returns a Process that will Close() when any given signal fires.
// This is useful to bind Process trees to syscall.SIGTERM, SIGKILL, etc.
func WithSignals(sig ...os.Signal) Process {
	p := WithParent(Background())
	c := make(chan os.Signal)
	signal.Notify(c, sig...)
	go func() {
		<-c
		signal.Stop(c)
		p.Close()
	}()
	return p
}

// Background returns the "background" Process: a statically allocated
// process that can _never_ close. It also never enters Closing() state.
// Calling Background().Close() will hang indefinitely.
func Background() Process {
	return background
}

// background is the background process
var background = &unclosable{Process: newProcess(nil)}

// unclosable is a process that _cannot_ be closed. calling Close simply hangs.
type unclosable struct {
	Process
}

func (p *unclosable) Close() error {
	var hang chan struct{}
	<-hang // hang forever
	return nil
}
