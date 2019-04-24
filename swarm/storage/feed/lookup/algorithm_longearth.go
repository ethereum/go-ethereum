package lookup

import (
	"context"
	"sync/atomic"
	"time"
)

type stepFunc func(ctx context.Context, t uint64, hint Epoch) interface{}

// LongEarthLookaheadDelay is the headstart the lookahead gives R before it launches
var LongEarthLookaheadDelay = 250 * time.Millisecond

// LongEarthLookbackDelay is the headstart the lookback gives R before it launches
var LongEarthLookbackDelay = 250 * time.Millisecond

// LongEarthAlgorithm explores possible lookup paths in parallel, pruning paths as soon
// as a more promising lookup path is found. As a result, this lookup algorithm is an order
// of magnitude faster than the FluzCapacitor algorithm, but at the expense of more exploratory reads.
// This algorithm works as follows. On each step, the next epoch is immediately looked up (R)
// and given a head start, while two parallel "steps" are launched a short time after:
// look ahead (A) is the path the algorithm would take if the R lookup returns a value, whereas
// look back (B) is the path the algorithm would take if the R lookup failed.
// as soon as R is actually finished, the A or B paths are pruned depending on the value of R.
// if A returns earlier than R, then R and B read operations can be safely canceled, saving time.
// The maximum number of active read operations is calculated as 2^(timeout/headstart).
// If headstart is infinite, this algorithm behaves as FluzCapacitor.
// timeout is the maximum execution time of the passed `read` function.
// the two head starts can be configured by changing LongEarthLookaheadDelay or LongEarthLookbackDelay
func LongEarthAlgorithm(ctx context.Context, now uint64, hint Epoch, read ReadFunc) (interface{}, error) {
	if hint == NoClue {
		hint = worstHint
	}

	var stepCounter int32 // for debugging, stepCounter allows to give an ID to each step instance

	errc := make(chan struct{}) // errc will help as an error shortcut signal
	var gerr error              // in case of error, this variable will be set

	var step stepFunc // For efficiency, the algorithm step is defined as a closure
	step = func(ctxS context.Context, t uint64, last Epoch) interface{} {
		stepID := atomic.AddInt32(&stepCounter, 1) // give an ID to this call instance
		trace(stepID, "init: t=%d, last=%s", t, last.String())
		var valueA, valueB, valueR interface{}

		// initialize the three read contexts
		ctxR, cancelR := context.WithCancel(ctxS) // will handle the current read operation
		ctxA, cancelA := context.WithCancel(ctxS) // will handle the lookahead path
		ctxB, cancelB := context.WithCancel(ctxS) // will handle the lookback path

		epoch := GetNextEpoch(last, t) // calculate the epoch to look up in this step instance

		// define the lookAhead function, which will follow the path as if R was successful
		lookAhead := func() {
			valueA = step(ctxA, t, epoch) // launch the next step, recursively.
			if valueA != nil {            // if this path is successful, we don't need R or B.
				cancelB()
				cancelR()
			}
		}

		// define the lookBack function, which will follow the path as if R was unsuccessful
		lookBack := func() {
			if epoch.Base() == last.Base() {
				return
			}
			base := epoch.Base()
			if base == 0 {
				return
			}
			valueB = step(ctxB, base-1, last)
		}

		go func() { //goroutine to read the current epoch (R)
			defer cancelR()
			var err error
			valueR, err = read(ctxR, epoch, now) // read this epoch
			if valueR == nil {                   // if unsuccessful, cancel lookahead, otherwise cancel lookback.
				cancelA()
			} else {
				cancelB()
			}
			if err != nil && err != context.Canceled {
				gerr = err
				close(errc)
			}
		}()

		go func() { // goroutine to give a headstart to R and then launch lookahead.
			defer cancelA()

			// if we are at the lowest level or the epoch to look up equals the last one,
			// then we cannot lookahead (can't go lower or repeat the same lookup, this would
			// cause an infinite loop)
			if epoch.Level == LowestLevel || epoch.Equals(last) {
				return
			}

			// give a head start to R, or launch immediately if R finishes early enough
			select {
			case <-TimeAfter(LongEarthLookaheadDelay):
				lookAhead()
			case <-ctxR.Done():
				if valueR != nil {
					lookAhead() // only look ahead if R was successful
				}
			case <-ctxA.Done():
			}
		}()

		go func() { // goroutine to give a headstart to R and then launch lookback.
			defer cancelB()

			// give a head start to R, or launch immediately if R finishes early enough
			select {
			case <-TimeAfter(LongEarthLookbackDelay):
				lookBack()
			case <-ctxR.Done():
				if valueR == nil {
					lookBack() // only look back in case R failed
				}
			case <-ctxB.Done():
			}
		}()

		<-ctxA.Done()
		if valueA != nil {
			trace(stepID, "Returning valueA=%v", valueA)
			return valueA
		}

		<-ctxR.Done()
		if valueR != nil {
			trace(stepID, "Returning valueR=%v", valueR)
			return valueR
		}
		<-ctxB.Done()
		trace(stepID, "Returning valueB=%v", valueB)
		return valueB
	}

	var value interface{}
	stepCtx, cancel := context.WithCancel(ctx)

	go func() { // launch the root step in its own goroutine to allow cancellation
		defer cancel()
		value = step(stepCtx, now, hint)
	}()

	// wait for the algorithm to finish, but shortcut in case
	// of errors
	select {
	case <-stepCtx.Done():
	case <-errc:
		cancel()
		return nil, gerr
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if value != nil || hint == worstHint {
		return value, nil
	}

	// at this point the algorithm did not return a value,
	// so we challenge the hint given.
	value, err := read(ctx, hint, now)
	if err != nil {
		return nil, err
	}
	if value != nil {
		return value, nil // hint is valid, return it.
	}

	// hint is invalid. Invoke the algorithm
	// without hint.
	now = hint.Base()
	if hint.Level == HighestLevel {
		now--
	}

	return LongEarthAlgorithm(ctx, now, NoClue, read)
}
