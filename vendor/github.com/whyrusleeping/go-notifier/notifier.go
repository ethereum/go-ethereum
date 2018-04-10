// Package notifier provides a simple notification dispatcher
// meant to be embedded in larger structres who wish to allow
// clients to sign up for event notifications.
package notifier

import (
	"sync"

	process "github.com/jbenet/goprocess"
	ratelimit "github.com/jbenet/goprocess/ratelimit"
)

// Notifiee is a generic interface. Clients implement
// their own Notifiee interfaces to ensure type-safety
// of notifications:
//
//  type RocketNotifiee interface{
//    Countdown(r Rocket, countdown time.Duration)
//    LiftedOff(Rocket)
//    ReachedOrbit(Rocket)
//    Detached(Rocket, Capsule)
//    Landed(Rocket)
//  }
//
type Notifiee interface{}

// Notifier is a notification dispatcher. It's meant
// to be composed, and its zero-value is ready to be used.
//
//  type Rocket struct {
//    notifier notifier.Notifier
//  }
//
type Notifier struct {
	mu   sync.RWMutex // guards notifiees
	nots map[Notifiee]struct{}
	lim  *ratelimit.RateLimiter
}

// RateLimited returns a rate limited Notifier. only limit goroutines
// will be spawned. If limit is zero, no rate limiting happens. This
// is the same as `Notifier{}`.
func RateLimited(limit int) Notifier {
	n := Notifier{}
	if limit > 0 {
		n.lim = ratelimit.NewRateLimiter(process.Background(), limit)
	}
	return n
}

// Notify signs up Notifiee e for notifications. This function
// is meant to be called behind your own type-safe function(s):
//
//   // generic function for pattern-following
//   func (r *Rocket) Notify(n Notifiee) {
//     r.notifier.Notify(n)
//   }
//
//   // or as part of other functions
//   func (r *Rocket) Onboard(a Astronaut) {
//     r.astronauts = append(r.austronauts, a)
//     r.notifier.Notify(a)
//   }
//
func (n *Notifier) Notify(e Notifiee) {
	n.mu.Lock()
	if n.nots == nil { // so that zero-value is ready to be used.
		n.nots = make(map[Notifiee]struct{})
	}
	n.nots[e] = struct{}{}
	n.mu.Unlock()
}

// StopNotify stops notifying Notifiee e. This function
// is meant to be called behind your own type-safe function(s):
//
//   // generic function for pattern-following
//   func (r *Rocket) StopNotify(n Notifiee) {
//     r.notifier.StopNotify(n)
//   }
//
//   // or as part of other functions
//   func (r *Rocket) Detach(c Capsule) {
//     r.notifier.StopNotify(c)
//     r.capsule = nil
//   }
//
func (n *Notifier) StopNotify(e Notifiee) {
	n.mu.Lock()
	if n.nots != nil { // so that zero-value is ready to be used.
		delete(n.nots, e)
	}
	n.mu.Unlock()
}

// NotifyAll messages the notifier's notifiees with a given notification.
// This is done by calling the given function with each notifiee. It is
// meant to be called with your own type-safe notification functions:
//
//  func (r *Rocket) Launch() {
//    r.notifyAll(func(n Notifiee) {
//      n.Launched(r)
//    })
//  }
//
//  // make it private so only you can use it. This function is necessary
//  // to make sure you only up-cast in one place. You control who you added
//  // to be a notifiee. If Go adds generics, maybe we can get rid of this
//  // method but for now it is like wrapping a type-less container with
//  // a type safe interface.
//  func (r *Rocket) notifyAll(notify func(Notifiee)) {
//    r.notifier.NotifyAll(func(n notifier.Notifiee) {
//      notify(n.(Notifiee))
//    })
//  }
//
// Note well: each notification is launched in its own goroutine, so they
// can be processed concurrently, and so that whatever the notification does
// it _never_ blocks out the client. This is so that consumers _cannot_ add
// hooks into your object that block you accidentally.
func (n *Notifier) NotifyAll(notify func(Notifiee)) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.nots == nil { // so that zero-value is ready to be used.
		return
	}

	// no rate limiting.
	if n.lim == nil {
		for notifiee := range n.nots {
			go notify(notifiee)
		}
		return
	}

	// with rate limiting.
	n.lim.Go(func(worker process.Process) {
		for notifiee := range n.nots {
			notifiee := notifiee // rebind for loop data races
			n.lim.LimitedGo(func(worker process.Process) {
				notify(notifiee)
			})
		}
	})
}
