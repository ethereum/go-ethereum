// Copyright 2015 The go-ethereum Authors
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

package v2

import (
	"reflect"

	"sync"

	"github.com/ethereum/go-ethereum/event"
)

// callback is a method callback which was registered in the server
type callback struct {
	method      reflect.Method // callback
	argTypes    []reflect.Type // input argument types
	errPos      int            // err return idx, of -1 when method cannot return error
	isSubscribe bool           // indication if the callback is a subscription
}

// service represents a registered object
type service struct {
	name          string        // name for service
	rcvr          reflect.Value // receiver of methods for the service
	typ           reflect.Type  // receiver type
	callbacks     callbacks     // registered handlers
	subscriptions subscriptions // available subscriptions/notifications
}

// serverRequest is an incoming request
type serverRequest struct {
	id            int64
	svcname       string
	rcvr          reflect.Value
	callb         *callback
	args          []reflect.Value
	isUnsubscribe bool
	err           RPCError
}

type serviceRegistry map[string]*service          // collection of services
type callbacks map[string]*callback               // collection of RPC callbacks
type subscriptions map[string]*callback           // collection of subscription callbacks
type subscriptionRegistry map[string]Subscription // collection of subscriptions

// Server represents a RPC server
type Server struct {
	services       serviceRegistry
	muSubcriptions sync.Mutex // protects subscriptions
	subscriptions  subscriptionRegistry
}

// rpcRequest represents a raw incoming RPC request
type rpcRequest struct {
	service  string
	method   string
	id       int64
	isPubSub bool
	params   interface{}
}

// RPCError implements RPC error, is add support for error codec over regular go errors
type RPCError interface {
	// RPC error code
	Code() int
	// Error message
	Error() string
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	// Read next request
	ReadRequestHeaders() ([]rpcRequest, bool, RPCError)
	// Parse request argument to the given types
	ParseRequestArguments([]reflect.Type, interface{}) ([]reflect.Value, RPCError)
	// Assemble success response
	CreateResponse(int64, interface{}) interface{}
	// Assemble error response
	CreateErrorResponse(*int64, RPCError) interface{}
	// Assemble error response with extra information about the error through info
	CreateErrorResponseWithInfo(id *int64, err RPCError, info interface{}) interface{}
	// Create notification response
	CreateNotification(string, interface{}) interface{}
	// Write msg to client.
	Write(interface{}) error
	// Close underlying data stream
	Close()
	// Closed when underlying connection is closed
	Closed() <-chan interface{}
}

// SubscriptionMatcher returns true if the given value matches the criteria specified by the user
type SubscriptionMatcher func(interface{}) bool

// Subscription is used by the server to send notifications to the client
type Subscription struct {
	sub   event.Subscription
	match SubscriptionMatcher
}

// NewSubscription create a new RPC subscription
func NewSubscription(sub event.Subscription) Subscription {
	return Subscription{sub, nil}
}

// NewSubscriptionFiltered will create a new subscription. For each raised event the given matcher is
// called. If it returns true the event is send as notification to the client, otherwise it is ignored.
func NewSubscriptionFiltered(sub event.Subscription, match SubscriptionMatcher) Subscription {
	return Subscription{sub, match}
}

// Chan returns the channel where new events will be published. It's up the user to call the matcher to
// determine if the events are interesting for the client.
func (s *Subscription) Chan() <-chan *event.Event {
	return s.sub.Chan()
}

// Unsubscribe will end the subscription and closes the event channel
func (s *Subscription) Unsubscribe() {
	s.sub.Unsubscribe()
}
