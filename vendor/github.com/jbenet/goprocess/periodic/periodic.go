// Package periodic is part of github.com/jbenet/goprocess.
// It provides a simple periodic processor that calls a function
// periodically based on some options.
//
// For example:
//
//    // use a time.Duration
//    p := periodicproc.Every(time.Second, func(proc goprocess.Process) {
//    	fmt.Printf("the time is %s and all is well", time.Now())
//    })
//
//    <-time.After(5*time.Second)
//    p.Close()
//
//    // use a time.Time channel (like time.Ticker)
//    p := periodicproc.Tick(time.Tick(time.Second), func(proc goprocess.Process) {
//    	fmt.Printf("the time is %s and all is well", time.Now())
//    })
//
//    <-time.After(5*time.Second)
//    p.Close()
//
//    // or arbitrary signals
//    signal := make(chan struct{})
//    p := periodicproc.OnSignal(signal, func(proc goprocess.Process) {
//    	fmt.Printf("the time is %s and all is well", time.Now())
//    })
//
//    signal<- struct{}{}
//    signal<- struct{}{}
//    <-time.After(5 * time.Second)
//    signal<- struct{}{}
//    p.Close()
//
package periodicproc

import (
	"time"

	gp "github.com/jbenet/goprocess"
)

// Every calls the given ProcessFunc at periodic intervals. Internally, it uses
// <-time.After(interval), so it will have the behavior of waiting _at least_
// interval in between calls. If you'd prefer the time.Ticker behavior, use
// periodicproc.Tick instead.
// This is sequentially rate limited, only one call will be in-flight at a time.
func Every(interval time.Duration, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		for {
			select {
			case <-time.After(interval):
				select {
				case <-proc.Go(procfunc).Closed(): // spin it out as a child, and wait till it's done.
				case <-proc.Closing(): // we're told to close
					return
				}
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	})
}

// EveryGo calls the given ProcessFunc at periodic intervals. Internally, it uses
// <-time.After(interval)
// This is not rate limited, multiple calls could be in-flight at the same time.
func EveryGo(interval time.Duration, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		for {
			select {
			case <-time.After(interval):
				proc.Go(procfunc)
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	})
}

// Tick constructs a ticker with interval, and calls the given ProcessFunc every
// time the ticker fires.
// This is sequentially rate limited, only one call will be in-flight at a time.
//
//  p := periodicproc.Tick(time.Second, func(proc goprocess.Process) {
//  	fmt.Println("fire!")
//  })
//
//  <-time.After(3 * time.Second)
//  p.Close()
//
//  // Output:
//  // fire!
//  // fire!
//  // fire!
func Tick(interval time.Duration, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		ticker := time.NewTicker(interval)
		callOnTicker(ticker.C, procfunc)(proc)
		ticker.Stop()
	})
}

// TickGo constructs a ticker with interval, and calls the given ProcessFunc every
// time the ticker fires.
// This is not rate limited, multiple calls could be in-flight at the same time.
//
//  p := periodicproc.TickGo(time.Second, func(proc goprocess.Process) {
//  	fmt.Println("fire!")
//  	<-time.After(10 * time.Second) // will not block sequential execution
//  })
//
//  <-time.After(3 * time.Second)
//  p.Close()
//
//  // Output:
//  // fire!
//  // fire!
//  // fire!
func TickGo(interval time.Duration, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		ticker := time.NewTicker(interval)
		goCallOnTicker(ticker.C, procfunc)(proc)
		ticker.Stop()
	})
}

// Ticker calls the given ProcessFunc every time the ticker fires.
// This is sequentially rate limited, only one call will be in-flight at a time.
func Ticker(ticker <-chan time.Time, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(callOnTicker(ticker, procfunc))
}

// TickerGo calls the given ProcessFunc every time the ticker fires.
// This is not rate limited, multiple calls could be in-flight at the same time.
func TickerGo(ticker <-chan time.Time, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(goCallOnTicker(ticker, procfunc))
}

func callOnTicker(ticker <-chan time.Time, pf gp.ProcessFunc) gp.ProcessFunc {
	return func(proc gp.Process) {
		for {
			select {
			case <-ticker:
				select {
				case <-proc.Go(pf).Closed(): // spin it out as a child, and wait till it's done.
				case <-proc.Closing(): // we're told to close
					return
				}
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	}
}

func goCallOnTicker(ticker <-chan time.Time, pf gp.ProcessFunc) gp.ProcessFunc {
	return func(proc gp.Process) {
		for {
			select {
			case <-ticker:
				proc.Go(pf)
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	}
}

// OnSignal calls the given ProcessFunc every time the signal fires, and waits for it to exit.
// This is sequentially rate limited, only one call will be in-flight at a time.
//
//  sig := make(chan struct{})
//  p := periodicproc.OnSignal(sig, func(proc goprocess.Process) {
//  	fmt.Println("fire!")
//  	<-time.After(time.Second) // delays sequential execution by 1 second
//  })
//
//  sig<- struct{}
//  sig<- struct{}
//  sig<- struct{}
//
//  // Output:
//  // fire!
//  // fire!
//  // fire!
func OnSignal(sig <-chan struct{}, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		for {
			select {
			case <-sig:
				select {
				case <-proc.Go(procfunc).Closed(): // spin it out as a child, and wait till it's done.
				case <-proc.Closing(): // we're told to close
					return
				}
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	})
}

// OnSignalGo calls the given ProcessFunc every time the signal fires.
// This is not rate limited, multiple calls could be in-flight at the same time.
//
//  sig := make(chan struct{})
//  p := periodicproc.OnSignalGo(sig, func(proc goprocess.Process) {
//  	fmt.Println("fire!")
//  	<-time.After(time.Second) // wont block execution
//  })
//
//  sig<- struct{}
//  sig<- struct{}
//  sig<- struct{}
//
//  // Output:
//  // fire!
//  // fire!
//  // fire!
func OnSignalGo(sig <-chan struct{}, procfunc gp.ProcessFunc) gp.Process {
	return gp.Go(func(proc gp.Process) {
		for {
			select {
			case <-sig:
				proc.Go(procfunc)
			case <-proc.Closing(): // we're told to close
				return
			}
		}
	})
}
