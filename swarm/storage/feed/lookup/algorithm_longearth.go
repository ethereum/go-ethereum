package lookup

import (
	"context"
	"sync/atomic"
	"time"
)

type StepFunc func(ctx context.Context, t uint64, hint Epoch) interface{}

func LongEarthAlgorithm(ctx context.Context, now uint64, hint Epoch, read ReadFunc) (interface{}, error) {
	var stepCounter int32

	errc := make(chan struct{})
	var gerr error

	var step StepFunc
	step = func(ctxS context.Context, t uint64, hint Epoch) interface{} {
		stepID := atomic.AddInt32(&stepCounter, 1)
		trace(stepID, "init: t=%d, hint=%s", t, hint.String())
		var valueA, valueB, valueR interface{}

		ctxR, cancelR := context.WithCancel(ctxS)
		ctxA, cancelA := context.WithCancel(ctxS)
		ctxB, cancelB := context.WithCancel(ctxS)

		epoch := GetNextEpoch(hint, t)

		lookAhead := func() {
			valueA = step(ctxA, t, epoch)
			if valueA != nil {
				cancelB()
				cancelR()
			}
		}

		lookBack := func() {
			var err error
			if epoch.Base() == hint.Base() {
				// we have reached the hint itself
				if hint == worstHint {
					valueB = nil
					return
				}
				// check it out
				valueB, err = read(ctxB, hint, now)
				if valueB != nil || err == context.Canceled {
					return
				}
				if err != nil {
					gerr = err
					close(errc)
					return
				}
				// bad hint.
				valueB = step(ctxB, hint.Base(), worstHint)
				return
			}
			base := epoch.Base()
			if base == 0 {
				return
			}
			valueB = step(ctxB, base-1, hint)
		}

		go func() {
			defer cancelR()
			var err error
			valueR, err = read(ctxR, epoch, now)
			if valueR == nil {
				cancelA()
			} else {
				cancelB()
			}
			if err != nil && err != context.Canceled {
				gerr = err
				close(errc)
			}
		}()

		go func() {
			defer cancelA()

			if epoch.Level == LowestLevel || epoch.Equals(hint) {
				return
			}

			select {
			case <-TimeAfter(250 * time.Millisecond):
				lookAhead()
			case <-ctxR.Done():
				if valueR != nil {
					lookAhead()
				}
			case <-ctxA.Done():
			}
		}()

		go func() {
			defer cancelB()

			select {
			case <-TimeAfter(250 * time.Millisecond):
				lookBack()
			case <-ctxR.Done():
				if valueR == nil {
					lookBack()
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
	defer cancel()

	go func() {
		value = step(stepCtx, now, hint)
		cancel()
	}()

	select {
	case <-stepCtx.Done():
		return value, ctx.Err()
	case <-errc:
		return nil, gerr
	}
}
