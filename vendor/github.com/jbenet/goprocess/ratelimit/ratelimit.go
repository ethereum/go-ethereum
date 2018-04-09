// Package ratelimit is part of github.com/jbenet/goprocess.
// It provides a simple process that ratelimits child creation.
// This is done internally with a channel/semaphore.
// So the call `RateLimiter.LimitedGo` may block until another
// child is Closed().
package ratelimit

import (
	process "github.com/jbenet/goprocess"
)

// RateLimiter limits the spawning of children. It does so
// with an internal semaphore. Note that Go will continue
// to be the unlimited process.Process.Go, and ONLY the
// added function `RateLimiter.LimitedGo` will honor the
// limit. This is to improve readability and avoid confusion
// for the reader, particularly if code changes over time.
type RateLimiter struct {
	process.Process

	limiter chan struct{}
}

func NewRateLimiter(parent process.Process, limit int) *RateLimiter {
	proc := process.WithParent(parent)
	return &RateLimiter{Process: proc, limiter: LimitChan(limit)}
}

// LimitedGo creates a new process, adds it as a child, and spawns the
// ProcessFunc f in its own goroutine, but may block according to the
// internal rate limit. It is equivalent to:
//
//   func(f process.ProcessFunc) {
//      <-limitch
//      p.Go(func (child process.Process) {
//        f(child)
//        f.Close() // make sure its children close too!
//        limitch<- struct{}{}
//      })
///  }
//
// It is useful to construct simple asynchronous workers, children of p,
// and rate limit their creation, to avoid spinning up too many, too fast.
// This is great for providing backpressure to producers.
func (rl *RateLimiter) LimitedGo(f process.ProcessFunc) {

	<-rl.limiter
	p := rl.Go(f)

	// this <-closed() is here because the child may have spawned
	// children of its own, and our rate limiter should capture that.
	go func() {
		<-p.Closed()
		rl.limiter <- struct{}{}
	}()
}

// LimitChan returns a rate-limiting channel. it is the usual, simple,
// golang-idiomatic rate-limiting semaphore. This function merely
// initializes it with certain buffer size, and sends that many values,
// so it is ready to be used.
func LimitChan(limit int) chan struct{} {
	limitch := make(chan struct{}, limit)
	for i := 0; i < limit; i++ {
		limitch <- struct{}{}
	}
	return limitch
}
