# goprocess - lifecycles in go

[![travisbadge](https://travis-ci.org/jbenet/goprocess.svg)](https://travis-ci.org/jbenet/goprocess)

(Based on https://github.com/jbenet/go-ctxgroup)

- Godoc: https://godoc.org/github.com/jbenet/goprocess

`goprocess` introduces a way to manage process lifecycles in go. It is
much like [go.net/context](https://godoc.org/code.google.com/p/go.net/context)
(it actually uses a Context), but it is more like a Context-WaitGroup hybrid.
`goprocess` is about being able to start and stop units of work, which may
receive `Close` signals from many clients. Think of it like a UNIX process
tree, but inside go.

`goprocess` seeks to minimally affect your objects, so you can use it
with both embedding or composition. At the heart of `goprocess` is the
`Process` interface:

```Go
// Process is the basic unit of work in goprocess. It defines a computation
// with a lifecycle:
// - running (before calling Close),
// - closing (after calling Close at least once),
// - closed (after Close returns, and all teardown has _completed_).
//
// More specifically, it fits this:
//
//   p := WithTeardown(tf) // new process is created, it is now running.
//   p.AddChild(q)         // can register children **before** Closing.
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

  // Go creates a new process, adds it as a child, and spawns the ProcessFunc f
  // in its own goroutine. It is equivalent to:
  //
  //   GoChild(p, f)
  //
  // It is useful to construct simple asynchronous workers, children of p.
  Go(f ProcessFunc) Process

  // Close ends the process. Close blocks until the process has completely
  // shut down, and any teardown has run _exactly once_. The returned error
  // is available indefinitely: calling Close twice returns the same error.
  // If the process has already been closed, Close returns immediately.
  Close() error

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
}
```
