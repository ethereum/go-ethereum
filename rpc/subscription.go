// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrNotificationsUnsupported is returned when the connection doesn't support notifications
	ErrNotificationsUnsupported = errors.New("notifications not supported")
	// ErrNotificationNotFound is returned when the notification for the given id is not found
	ErrSubscriptionNotFound     = errors.New("subscription not found")
	ErrNamespaceNotFound        = errors.New("namespace not found")
	ErrSubscriptionTypeNotFound = errors.New("no active subscriptions of specified type")
)

// ID defines a pseudo random number that is used to identify RPC subscriptions.
type ID string

// SubscriptionType determines the type of subscription (which allows sending updates to
// subscriptions of a particular type); this is also used by eth/filters to index different
// filter types.
type SubscriptionType byte

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription SubscriptionType = iota
	// LogsSubscription queries for new or removed (chain reorg) logs
	LogsSubscription
	// PendingLogsSubscription queries for logs in pending blocks
	PendingLogsSubscription
	// MinedAndPendingLogsSubscription queries for logs in mined and pending blocks.
	MinedAndPendingLogsSubscription
	// PendingTransactionsSubscription queries tx hashes for pending
	// transactions entering the pending state
	PendingTransactionsSubscription
	// BlocksSubscription queries hashes for blocks that are imported
	BlocksSubscription
	// ReturnData queries for return data from transactions executed by a
	// particular rpc client
	ReturnDataSubscription
	// LastSubscription keeps track of the last index
	LastIndexSubscription
)

// a Subscription is created by a notifier and tied to that notifier. The client can use
// this subscription to wait for an unsubscribe request for the client, see Err().
type Subscription struct {
	ID        ID
	namespace string
	Type      SubscriptionType
	update    chan interface{} // used to send update filter criteria of subscription
	err       chan error       // closed on unsubscribe
}

func (s *Subscription) SetType(subType SubscriptionType) {
	s.Type = subType
}

func (s *Subscription) Update() <-chan interface{} {
	return s.update
}

// Err returns a channel that is closed when the client sends an unsubscribe request.
func (s *Subscription) Err() <-chan error {
	return s.err
}

// notifierKey is used to store a notifier within the connection context.
type notifierKey struct{}

// Notifier is tied to a RPC connection that supports subscriptions.
// Server callbacks use the notifier to send notifications.
type Notifier struct {
	codec    ServerCodec
	subMu    sync.RWMutex // guards active and inactive maps
	active   map[ID]*Subscription
	inactive map[ID]*Subscription
}

// newNotifier creates a new notifier that can be used to send subscription
// notifications to the client.
func newNotifier(codec ServerCodec) *Notifier {
	return &Notifier{
		codec:    codec,
		active:   make(map[ID]*Subscription),
		inactive: make(map[ID]*Subscription),
	}
}

// NotifierFromContext returns the Notifier value stored in ctx, if any.
func NotifierFromContext(ctx context.Context) (*Notifier, bool) {
	n, ok := ctx.Value(notifierKey{}).(*Notifier)
	return n, ok
}

// CreateSubscription returns a new subscription that is coupled to the
// RPC connection. By default subscriptions are inactive and notifications
// are dropped until the subscription is marked as active. This is done
// by the RPC server after the subscription ID is send to the client.
func (n *Notifier) CreateSubscription() *Subscription {
	s := &Subscription{ID: NewID(), update: make(chan interface{}), err: make(chan error)}
	n.subMu.Lock()
	n.inactive[s.ID] = s
	n.subMu.Unlock()
	return s
}

// Notify sends a notification to the client with the given data as payload.
// If an error occurs the RPC connection is closed and the error is returned.
func (n *Notifier) Notify(id ID, data interface{}) error {
	n.subMu.RLock()
	defer n.subMu.RUnlock()

	sub, active := n.active[id]
	if active {
		notification := n.codec.CreateNotification(string(id), sub.namespace, data)
		if err := n.codec.Write(notification); err != nil {
			n.codec.Close()
			return err
		}
	}
	return nil
}

// Send an update message to the update channel of a particular subscription id.
//
// Intended use pattern is to update the filter criteria about which events should
//   generate notifcations.
//
func (n *Notifier) UpdateSubscription(id ID, message interface{}) error {
	n.subMu.RLock()
	defer n.subMu.RUnlock()
	sub, validid := n.active[id]
	if validid {
		sub.update <- message
		return nil
	} else {
		return ErrSubscriptionNotFound
	}
}

// Send an update message to the update channels of all subscriptions of
//  the specified subscription type.
//
// Intended use pattern is to update the filter criteria about which events should
//   generate notifcations.  This version is especially useful if the subscription
//   id is not known by the caller.  Unlike UpdateScription, this can block.
//
//  For example, when a client is subscribed to ReturnData, each time the client
//   submits a new transaction, this is used to add the tx to the list of transactions
//   which should generate a TransactionEvent and notify the client with return data.
func (n *Notifier) UpdateSubscriptions(subType SubscriptionType, message interface{}) error {
	n.subMu.RLock()
	defer n.subMu.RUnlock()
	for _, sub := range n.active {
		if sub.Type == subType {
			sub.update <- message
			return nil
		}
	}

	return ErrSubscriptionTypeNotFound
}

// Closed returns a channel that is closed when the RPC connection is closed.
func (n *Notifier) Closed() <-chan interface{} {
	return n.codec.Closed()
}

// unsubscribe a subscription.
// If the subscription could not be found ErrSubscriptionNotFound is returned.
func (n *Notifier) unsubscribe(id ID) error {
	n.subMu.Lock()
	defer n.subMu.Unlock()
	if s, found := n.active[id]; found {
		close(s.err)
		delete(n.active, id)
		return nil
	}
	return ErrSubscriptionNotFound
}

// activate enables a subscription. Until a subscription is enabled all
// notifications are dropped. This method is called by the RPC server after
// the subscription ID was sent to client. This prevents notifications being
// send to the client before the subscription ID is send to the client.
func (n *Notifier) activate(id ID, namespace string) {
	n.subMu.Lock()
	defer n.subMu.Unlock()
	if sub, found := n.inactive[id]; found {
		sub.namespace = namespace
		n.active[id] = sub
		delete(n.inactive, id)
	}
}
