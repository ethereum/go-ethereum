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
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	// ErrNotificationsUnsupported is returned when the connection doesn't support notifications
	ErrNotificationsUnsupported = errors.New("notifications not supported")

	// ErrNotificationNotFound is returned when the notification for the given id is not found
	ErrNotificationNotFound = errors.New("notification not found")

	// errNotifierStopped is returned when the notifier is stopped (e.g. codec is closed)
	errNotifierStopped = errors.New("unable to send notification")

	// errNotificationQueueFull is returns when there are too many notifications in the queue
	errNotificationQueueFull = errors.New("too many pending notifications")
)

// unsubSignal is a signal that the subscription is unsubscribed. It is used to flush buffered
// notifications that might be pending in the internal queue.
var unsubSignal = new(struct{})

// UnsubscribeCallback defines a callback that is called when a subcription ends.
// It receives the subscription id as argument.
type UnsubscribeCallback func(id string)

// notification is a helper object that holds event data for a subscription
type notification struct {
	sub  *bufferedSubscription // subscription id
	data interface{}           // event data
}

// A Notifier type describes the interface for objects that can send create subscriptions
type Notifier interface {
	// Create a new subscription. The given callback is called when this subscription
	// is cancelled (e.g. client send an unsubscribe, connection closed).
	NewSubscription(UnsubscribeCallback) (Subscription, error)
	// Cancel subscription
	Unsubscribe(id string) error
}

// Subscription defines the interface for objects that can notify subscribers
type Subscription interface {
	// Inform client of an event
	Notify(data interface{}) error
	// Unique identifier
	ID() string
	// Cancel subscription
	Cancel() error
}

// bufferedSubscription is a subscription that uses a bufferedNotifier to send
// notifications to subscribers.
type bufferedSubscription struct {
	id               string
	unsubOnce        sync.Once           // call unsub method once
	unsub            UnsubscribeCallback // called on Unsubscribed
	notifier         *bufferedNotifier   // forward notifications to
	pending          chan interface{}    // closed when active
	flushed          chan interface{}    // closed when all buffered notifications are send
	lastNotification time.Time           // last time a notification was send
}

// ID returns the subscription identifier that the client uses to refer to this instance.
func (s *bufferedSubscription) ID() string {
	return s.id
}

// Cancel informs the notifier that this subscription is cancelled by the API
func (s *bufferedSubscription) Cancel() error {
	return s.notifier.Unsubscribe(s.id)
}

// Notify the subscriber of a particular event.
func (s *bufferedSubscription) Notify(data interface{}) error {
	return s.notifier.send(s.id, data)
}

// bufferedNotifier is a notifier that queues notifications in an internal queue and
// send them as fast as possible to the client from this queue. It will stop if the
// queue grows past a given size.
type bufferedNotifier struct {
	codec         ServerCodec                      // underlying connection
	mu            sync.Mutex                       // guard internal state
	subscriptions map[string]*bufferedSubscription // keep track of subscriptions associated with codec
	queueSize     int                              // max number of items in queue
	queue         chan *notification               // notification queue
	stopped       bool                             // indication if this notifier is ordered to stop
}

// newBufferedNotifier returns a notifier that queues notifications in an internal queue
// from which notifications are send as fast as possible to the client. If the queue size
// limit is reached (client is unable to keep up) it will stop and closes the codec.
func newBufferedNotifier(codec ServerCodec, size int) *bufferedNotifier {
	notifier := &bufferedNotifier{
		codec:         codec,
		subscriptions: make(map[string]*bufferedSubscription),
		queue:         make(chan *notification, size),
		queueSize:     size,
	}

	go notifier.run()

	return notifier
}

// NewSubscription creates a new subscription that forwards events to this instance internal
// queue. The given callback is called when the subscription is unsubscribed/cancelled.
func (n *bufferedNotifier) NewSubscription(callback UnsubscribeCallback) (Subscription, error) {
	id, err := newSubscriptionID()
	if err != nil {
		return nil, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.stopped {
		return nil, errNotifierStopped
	}

	sub := &bufferedSubscription{
		id:               id,
		unsub:            callback,
		notifier:         n,
		pending:          make(chan interface{}),
		flushed:          make(chan interface{}),
		lastNotification: time.Now(),
	}

	n.subscriptions[id] = sub

	return sub, nil
}

// Remove the given subscription. If subscription is not found notificationNotFoundErr is returned.
func (n *bufferedNotifier) Unsubscribe(subid string) error {
	n.mu.Lock()
	sub, found := n.subscriptions[subid]
	n.mu.Unlock()

	if found {
		// send the unsubscribe signal, this will cause the notifier not to accept new events
		// for this subscription and will close the flushed channel after the last (buffered)
		// notification was send to the client.
		if err := n.send(subid, unsubSignal); err != nil {
			return err
		}

		// wait for confirmation that all (buffered) events are send for this subscription.
		// this ensures that the unsubscribe method response is not send before all buffered
		// events for this subscription are send.
		<-sub.flushed

		return nil
	}

	return ErrNotificationNotFound
}

// Send enques the given data for the subscription with public ID on the internal queue. t returns
// an error when the notifier is stopped or the queue is full. If data is the unsubscribe signal it
// will remove the subscription with the given id from the subscription collection.
func (n *bufferedNotifier) send(id string, data interface{}) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.stopped {
		return errNotifierStopped
	}

	var (
		subscription *bufferedSubscription
		found        bool
	)

	// check if subscription is associated with this connection, it might be cancelled
	// (subscribe/connection closed)
	if subscription, found = n.subscriptions[id]; !found {
		glog.V(logger.Error).Infof("received notification for unknown subscription %s\n", id)
		return ErrNotificationNotFound
	}

	// received the unsubscribe signal. Add it to the queue to make sure any pending notifications
	// for this subscription are send. When the run loop receives this singal it will signal that
	// all pending subscriptions are flushed and that the confirmation of the unsubscribe can be
	// send to the user. Remove the subscriptions to make sure new notifications are not accepted.
	if data == unsubSignal {
		delete(n.subscriptions, id)
		if subscription.unsub != nil {
			subscription.unsubOnce.Do(func() { subscription.unsub(id) })
		}
	}

	subscription.lastNotification = time.Now()

	if len(n.queue) >= n.queueSize {
		glog.V(logger.Warn).Infoln("too many buffered notifications -> close connection")
		n.codec.Close()
		return errNotificationQueueFull
	}

	n.queue <- &notification{subscription, data}
	return nil
}

// run reads notifications from the internal queue and sends them to the client. In case of an
// error, or when the codec is closed it will cancel all active subscriptions and returns.
func (n *bufferedNotifier) run() {
	defer func() {
		n.mu.Lock()
		defer n.mu.Unlock()

		n.stopped = true
		close(n.queue)

		// on exit call unsubscribe callback
		for id, sub := range n.subscriptions {
			if sub.unsub != nil {
				sub.unsubOnce.Do(func() { sub.unsub(id) })
			}
			close(sub.flushed)
			delete(n.subscriptions, id)
		}
	}()

	for {
		select {
		case notification := <-n.queue:
			// It can happen that an event is raised before the RPC server was able to send the sub
			// id to the client. Therefore subscriptions are marked as pending until the sub id was
			// send. The RPC server will activate the subscription by closing the pending chan.
			<-notification.sub.pending

			if notification.data == unsubSignal {
				// unsubSignal is the last accepted message for this subscription. Raise the signal
				// that all buffered notifications are sent by closing the flushed channel. This
				// indicates that the response for the unsubscribe can be send to the client.
				close(notification.sub.flushed)
			} else {
				msg := n.codec.CreateNotification(notification.sub.id, notification.data)
				if err := n.codec.Write(msg); err != nil {
					n.codec.Close()
					// unable to send notification to client, unsubscribe all subscriptions
					glog.V(logger.Warn).Infof("unable to send notification - %v\n", err)
					return
				}
			}
		case <-n.codec.Closed(): // connection was closed
			glog.V(logger.Debug).Infoln("codec closed, stop subscriptions")
			return
		}
	}
}

// Marks the subscription as active. This will causes the notifications for this subscription to be
// forwarded to the client.
func (n *bufferedNotifier) activate(subid string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if sub, found := n.subscriptions[subid]; found {
		close(sub.pending)
	}
}
