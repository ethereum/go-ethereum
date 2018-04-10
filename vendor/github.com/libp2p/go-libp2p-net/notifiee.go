package net

import (
	ma "github.com/multiformats/go-multiaddr"
)

// NotifyBundle implements Notifiee by calling any of the functions set on it,
// and nop'ing if they are unset. This is the easy way to register for
// notifications.
type NotifyBundle struct {
	ListenF      func(Network, ma.Multiaddr)
	ListenCloseF func(Network, ma.Multiaddr)

	ConnectedF    func(Network, Conn)
	DisconnectedF func(Network, Conn)

	OpenedStreamF func(Network, Stream)
	ClosedStreamF func(Network, Stream)
}

var _ Notifiee = (*NotifyBundle)(nil)

func (nb *NotifyBundle) Listen(n Network, a ma.Multiaddr) {
	if nb.ListenF != nil {
		nb.ListenF(n, a)
	}
}

func (nb *NotifyBundle) ListenClose(n Network, a ma.Multiaddr) {
	if nb.ListenCloseF != nil {
		nb.ListenCloseF(n, a)
	}
}

func (nb *NotifyBundle) Connected(n Network, c Conn) {
	if nb.ConnectedF != nil {
		nb.ConnectedF(n, c)
	}
}

func (nb *NotifyBundle) Disconnected(n Network, c Conn) {
	if nb.DisconnectedF != nil {
		nb.DisconnectedF(n, c)
	}
}

func (nb *NotifyBundle) OpenedStream(n Network, s Stream) {
	if nb.OpenedStreamF != nil {
		nb.OpenedStreamF(n, s)
	}
}

func (nb *NotifyBundle) ClosedStream(n Network, s Stream) {
	if nb.ClosedStreamF != nil {
		nb.ClosedStreamF(n, s)
	}
}
