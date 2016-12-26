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
	id            interface{}
	svcname       string
	rcvr          reflect.Value
	callb         *callback
	args          []reflect.Value
	isUnsubscribe bool
	err           Error
}

type serviceRegistry map[string]*service       // collection of services
type callbacks map[string]*callback            // collection of RPC callbacks
type subscriptions map[string]*callback        // collection of subscription callbacks
type subscriptionRegistry map[string]*callback // collection of subscription callbacks

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
	id       interface{}
	isPubSub bool
	params   interface{}
	err      Error // invalid batch element
}

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	// Read next request
	ReadRequestHeaders() ([]rpcRequest, bool, Error)
	// Parse request argument to the given types
	ParseRequestArguments([]reflect.Type, interface{}) ([]reflect.Value, Error)
	// Assemble success response, expects response id and payload
	CreateResponse(interface{}, interface{}) interface{}
	// Assemble error response, expects response id and error
	CreateErrorResponse(interface{}, Error) interface{}
	// Assemble error response with extra information about the error through info
	CreateErrorResponseWithInfo(id interface{}, err Error, info interface{}) interface{}
	// Create notification response
	CreateNotification(string, interface{}) interface{}
	// Write msg to client.
	Write(interface{}) error
	// Close underlying data stream
	Close()
	// Closed when underlying connection is closed
	Closed() <-chan interface{}
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
// - "latest", "earliest" or "pending" as string arguments
// - the block number
// Returned errors:
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

func (bn BlockNumber) Int64() int64 {
	return (int64)(bn)
}
