package nat

import (
	notifier "github.com/whyrusleeping/go-notifier"
)

// Notifier is an object that assists NAT in notifying listeners.
// It is implemented using thirdparty/notifier
type Notifier struct {
	n notifier.Notifier
}

func (n *Notifier) notifyAll(notify func(n Notifiee)) {
	n.n.NotifyAll(func(n notifier.Notifiee) {
		notify(n.(Notifiee))
	})
}

// Notify signs up notifiee to listen to NAT events.
func (n *Notifier) Notify(notifiee Notifiee) {
	n.n.Notify(n)
}

// StopNotify stops signaling events to notifiee.
func (n *Notifier) StopNotify(notifiee Notifiee) {
	n.n.StopNotify(notifiee)
}

// Notifiee is an interface objects must implement to listen to NAT events.
type Notifiee interface {

	// Called every time a successful mapping happens
	// Warning: the port mapping may have changed. If that is the
	// case, both MappingSuccess and MappingChanged are called.
	MappingSuccess(nat *NAT, m Mapping)

	// Called when mapping a port succeeds, but the mapping is
	// with a different port than an earlier success.
	MappingChanged(nat *NAT, m Mapping, oldport int, newport int)

	// Called when a port mapping fails. NAT will continue attempting after
	// the next period. To stop trying, use: mapping.Close(). After this failure,
	// mapping.ExternalPort() will be zero, and nat.ExternalAddrs() will not
	// return the address for this mapping. With luck, the next attempt will
	// succeed, without the client needing to do anything.
	MappingFailed(nat *NAT, m Mapping, oldport int, err error)
}
