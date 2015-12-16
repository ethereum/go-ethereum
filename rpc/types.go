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

package rpc

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"gopkg.in/fatih/set.v0"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// callback is a method callback which was registered in the server
type callback struct {
	rcvr        reflect.Value  // receiver of method
	method      reflect.Method // callback
	argTypes    []reflect.Type // input argument types
	hasCtx      bool           // method's first argument is a context (not included in argTypes)
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

	run      int32
	codecsMu sync.Mutex
	codecs   *set.Set
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

// SubscriptionOutputFormat accepts event data and has the ability to format the data before it is send to the client
type SubscriptionOutputFormat func(interface{}) interface{}

// defaultSubscriptionOutputFormatter returns data and is used as default output format for notifications
func defaultSubscriptionOutputFormatter(data interface{}) interface{} {
	return data
}

// Subscription is used by the server to send notifications to the client
type Subscription struct {
	sub    event.Subscription
	match  SubscriptionMatcher
	format SubscriptionOutputFormat
}

// NewSubscription create a new RPC subscription
func NewSubscription(sub event.Subscription) Subscription {
	return Subscription{sub, nil, defaultSubscriptionOutputFormatter}
}

// NewSubscriptionWithOutputFormat create a new RPC subscription which a custom notification output format
func NewSubscriptionWithOutputFormat(sub event.Subscription, formatter SubscriptionOutputFormat) Subscription {
	return Subscription{sub, nil, formatter}
}

// NewSubscriptionFiltered will create a new subscription. For each raised event the given matcher is
// called. If it returns true the event is send as notification to the client, otherwise it is ignored.
func NewSubscriptionFiltered(sub event.Subscription, match SubscriptionMatcher) Subscription {
	return Subscription{sub, match, defaultSubscriptionOutputFormatter}
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

// HexNumber serializes a number to hex format using the "%#x" format
type HexNumber big.Int

// NewHexNumber creates a new hex number instance which will serialize the given val with `%#x` on marshal.
func NewHexNumber(val interface{}) *HexNumber {
	if val == nil {
		return nil
	}

	if v, ok := val.(*big.Int); ok && v != nil {
		hn := new(big.Int).Set(v)
		return (*HexNumber)(hn)
	}

	rval := reflect.ValueOf(val)

	var unsigned uint64
	utype := reflect.TypeOf(unsigned)
	if t := rval.Type(); t.ConvertibleTo(utype) {
		hn := new(big.Int).SetUint64(rval.Convert(utype).Uint())
		return (*HexNumber)(hn)
	}

	var signed int64
	stype := reflect.TypeOf(signed)
	if t := rval.Type(); t.ConvertibleTo(stype) {
		hn := new(big.Int).SetInt64(rval.Convert(stype).Int())
		return (*HexNumber)(hn)
	}

	return nil
}

func (h *HexNumber) UnmarshalJSON(input []byte) error {
	length := len(input)
	if length >= 2 && input[0] == '"' && input[length-1] == '"' {
		input = input[1 : length-1]
	}

	hn := (*big.Int)(h)
	if _, ok := hn.SetString(string(input), 0); ok {
		return nil
	}

	return fmt.Errorf("Unable to parse number")
}

// MarshalJSON serialize the hex number instance to a hex representation.
func (h *HexNumber) MarshalJSON() ([]byte, error) {
	if h != nil {
		hn := (*big.Int)(h)
		if hn.BitLen() == 0 {
			return []byte(`"0x0"`), nil
		}
		return []byte(fmt.Sprintf(`"0x%x"`, hn)), nil
	}
	return nil, nil
}

func (h *HexNumber) Int() int {
	hn := (*big.Int)(h)
	return int(hn.Int64())
}

func (h *HexNumber) Int64() int64 {
	hn := (*big.Int)(h)
	return hn.Int64()
}

func (h *HexNumber) Uint() uint {
	hn := (*big.Int)(h)
	return uint(hn.Uint64())
}

func (h *HexNumber) Uint64() uint64 {
	hn := (*big.Int)(h)
	return hn.Uint64()
}

func (h *HexNumber) BigInt() *big.Int {
	return (*big.Int)(h)
}

type Number int64

func (n *Number) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))

	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	if len(input) == 0 {
		*n = Number(latestBlockNumber.Int64())
		return nil
	}

	in := new(big.Int)
	_, ok := in.SetString(input, 0)

	if !ok { // test if user supplied string tag
		return fmt.Errorf(`invalid number %s`, data)
	}

	if in.Cmp(earliestBlockNumber) >= 0 && in.Cmp(maxBlockNumber) <= 0 {
		*n = Number(in.Int64())
		return nil
	}

	return fmt.Errorf("blocknumber not in range [%d, %d]", earliestBlockNumber, maxBlockNumber)
}

func (n *Number) Int64() int64 {
	return *(*int64)(n)
}

func (n *Number) BigInt() *big.Int {
	return big.NewInt(n.Int64())
}

var (
	pendingBlockNumber  = big.NewInt(-2)
	latestBlockNumber   = big.NewInt(-1)
	earliestBlockNumber = big.NewInt(0)
	maxBlockNumber      = big.NewInt(math.MaxInt64)
)

type BlockNumber int64

const (
	PendingBlockNumber = BlockNumber(-2)
	LatestBlockNumber  = BlockNumber(-1)
)

// UnmarshalJSON parses the given JSON fragement into a BlockNumber. It supports:
// - "latest" or "earliest" as string arguments
// - the block number
// Returned errors:
// - an unsupported error when "pending" is specified (not yet implemented)
// - an invalid block number error when the given argument isn't a known strings
// - an out of range error when the given block number is either too little or too large
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))

	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	if len(input) == 0 {
		*bn = BlockNumber(latestBlockNumber.Int64())
		return nil
	}

	in := new(big.Int)
	_, ok := in.SetString(input, 0)

	if !ok { // test if user supplied string tag
		strBlockNumber := input
		if strBlockNumber == "latest" {
			*bn = BlockNumber(latestBlockNumber.Int64())
			return nil
		}

		if strBlockNumber == "earliest" {
			*bn = BlockNumber(earliestBlockNumber.Int64())
			return nil
		}

		if strBlockNumber == "pending" {
			*bn = BlockNumber(pendingBlockNumber.Int64())
			return nil
		}

		return fmt.Errorf(`invalid blocknumber %s`, data)
	}

	if in.Cmp(earliestBlockNumber) >= 0 && in.Cmp(maxBlockNumber) <= 0 {
		*bn = BlockNumber(in.Int64())
		return nil
	}

	return fmt.Errorf("blocknumber not in range [%d, %d]", earliestBlockNumber, maxBlockNumber)
}

func (bn *BlockNumber) Int64() int64 {
	return (int64)(*bn)
}

// Client defines the interface for go client that wants to connect to a geth RPC endpoint
type Client interface {
	// SupportedModules returns the collection of API's the server offers
	SupportedModules() (map[string]string, error)

	Send(req interface{}) error
	Recv(msg interface{}) error

	Close()
}
