// This file extends the LesApiBackend with functions for BlockNative's dropped
// transaction feeds.
//
// As this is part of a fork, and not included in core geth, keeping it in a
// separate file helps protect against potential merge conflicts. If this were
// ever to be merged into core geth, it should be relocated to ./api_backend.go

package les

import (
	"errors"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
)

type unimplementedSubscription struct{}

func (s *unimplementedSubscription) Unsubscribe() {}
func (s *unimplementedSubscription) Err() <-chan error {
	ch := make(chan error, 1)
	ch <- errors.New("Subscription type not implemented")
	return ch
}

func (b *LesApiBackend) SubscribeDropTxsEvent(ch chan<- core.DropTxsEvent) event.Subscription {
	return &unimplementedSubscription{}
}

func (b *LesApiBackend) SubscribeRejectedTxEvent(ch chan<- core.RejectedTxEvent) event.Subscription {
	return &unimplementedSubscription{}
}
