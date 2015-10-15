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
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	jsonRPCVersion         = "2.0"
	serviceMethodSeparator = "_"
	subscribeMethod        = "eth_subscribe"
	unsubscribeMethod      = "eth_unsubscribe"
	notificationMethod     = "eth_subscription"
)

// JSON-RPC request
type jsonRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	Id      *int64          `json:"id,omitempty"`
	Payload json.RawMessage `json:"params"`
}

// JSON-RPC response
type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      int64       `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

// JSON-RPC error object
type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC error response
type jsonErrResponse struct {
	Version string    `json:"jsonrpc"`
	Id      *int64    `json:"id,omitempty"`
	Error   jsonError `json:"error"`
}

// JSON-RPC notification payload
type jsonSubscription struct {
	Subscription string      `json:"subscription"`
	Result       interface{} `json:"result,omitempty"`
}

// JSON-RPC notification
type jsonNotification struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  jsonSubscription `json:"params"`
}

// jsonCodec reads and writes JSON-RPC messages to the underlying connection. It also has support for parsing arguments
// and serializing (result) objects.
type jsonCodec struct {
	closed   chan interface{}
	isClosed int32
	d        *json.Decoder
	e        *json.Encoder
	req      jsonRequest
	rw       io.ReadWriteCloser
}

// NewJSONCodec creates a new RPC server codec with support for JSON-RPC 2.0
func NewJSONCodec(rwc io.ReadWriteCloser) ServerCodec {
	d := json.NewDecoder(rwc)
	d.UseNumber()
	return &jsonCodec{closed: make(chan interface{}), d: d, e: json.NewEncoder(rwc), rw: rwc, isClosed: 0}
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(msg json.RawMessage) bool {
	for _, c := range msg {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// ReadRequestHeaders will read new requests without parsing the arguments. It will return a collection of requests, an
// indication if these requests are in batch form or an error when the incoming message could not be read/parsed.
func (c *jsonCodec) ReadRequestHeaders() ([]rpcRequest, bool, RPCError) {
	var incomingMsg json.RawMessage
	if err := c.d.Decode(&incomingMsg); err != nil {
		return nil, false, &invalidRequestError{err.Error()}
	}

	if isBatch(incomingMsg) {
		return parseBatchRequest(incomingMsg)
	}

	return parseRequest(incomingMsg)
}

// parseRequest will parse a single request from the given RawMessage. It will return the parsed request, an indication
// if the request was a batch or an error when the request could not be parsed.
func parseRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, RPCError) {
	var in jsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	if in.Id == nil {
		return nil, false, &invalidMessageError{"Server cannot handle notifications"}
	}

	// subscribe are special, they will always use `subscribeMethod` as service method
	if in.Method == subscribeMethod {
		reqs := []rpcRequest{rpcRequest{id: *in.Id, isPubSub: true}}
		if len(in.Payload) > 0 {
			// first param must be subscription name
			var subscribeMethod [1]string
			if err := json.Unmarshal(in.Payload, &subscribeMethod); err != nil {
				glog.V(logger.Debug).Infof("Unable to parse subscription method: %v\n", err)
				return nil, false, &invalidRequestError{"Unable to parse subscription request"}
			}

			// all subscriptions are made on the eth service
			reqs[0].service, reqs[0].method = "eth", subscribeMethod[0]
			reqs[0].params = in.Payload
			return reqs, false, nil
		}
		return nil, false, &invalidRequestError{"Unable to parse subscription request"}
	}

	if in.Method == unsubscribeMethod {
		return []rpcRequest{rpcRequest{id: *in.Id, isPubSub: true,
			method: unsubscribeMethod, params: in.Payload}}, false, nil
	}

	// regular RPC call
	elems := strings.Split(in.Method, serviceMethodSeparator)
	if len(elems) != 2 {
		return nil, false, &methodNotFoundError{in.Method, ""}
	}

	if len(in.Payload) == 0 {
		return []rpcRequest{rpcRequest{service: elems[0], method: elems[1], id: *in.Id}}, false, nil
	}

	return []rpcRequest{rpcRequest{service: elems[0], method: elems[1], id: *in.Id, params: in.Payload}}, false, nil
}

// parseBatchRequest will parse a batch request into a collection of requests from the given RawMessage, an indication
// if the request was a batch or an error when the request could not be read.
func parseBatchRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, RPCError) {
	var in []jsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	requests := make([]rpcRequest, len(in))
	for i, r := range in {
		if r.Id == nil {
			return nil, true, &invalidMessageError{"Server cannot handle notifications"}
		}

		// (un)subscribe are special, they will always use the same service.method
		if r.Method == subscribeMethod {
			requests[i] = rpcRequest{id: *r.Id, isPubSub: true}
			if len(r.Payload) > 0 {
				var subscribeMethod [1]string
				if err := json.Unmarshal(r.Payload, &subscribeMethod); err != nil {
					glog.V(logger.Debug).Infof("Unable to parse subscription method: %v\n", err)
					return nil, false, &invalidRequestError{"Unable to parse subscription request"}
				}

				// all subscriptions are made on the eth service
				requests[i].service, requests[i].method = "eth", subscribeMethod[0]
				requests[i].params = r.Payload
				continue
			}

			return nil, true, &invalidRequestError{"Unable to parse (un)subscribe request arguments"}
		}

		if r.Method == unsubscribeMethod {
			requests[i] = rpcRequest{id: *r.Id, isPubSub: true, method: unsubscribeMethod, params: r.Payload}
			continue
		}

		elems := strings.Split(r.Method, serviceMethodSeparator)
		if len(elems) != 2 {
			return nil, true, &methodNotFoundError{r.Method, ""}
		}

		if len(r.Payload) == 0 {
			requests[i] = rpcRequest{service: elems[0], method: elems[1], id: *r.Id, params: nil}
		} else {
			requests[i] = rpcRequest{service: elems[0], method: elems[1], id: *r.Id, params: r.Payload}
		}
	}

	return requests, true, nil
}

// ParseRequestArguments tries to parse the given params (json.RawMessage) with the given types. It returns the parsed
// values or an error when the parsing failed.
func (c *jsonCodec) ParseRequestArguments(argTypes []reflect.Type, params interface{}) ([]reflect.Value, RPCError) {
	if args, ok := params.(json.RawMessage); !ok {
		return nil, &invalidParamsError{"Invalid params supplied"}
	} else {
		return parsePositionalArguments(args, argTypes)
	}
}

func countArguments(args json.RawMessage) (int, error) {
	var cnt []interface{}
	if err := json.Unmarshal(args, &cnt); err != nil {
		return -1, nil
	}
	return len(cnt), nil
}

// parsePositionalArguments tries to parse the given args to an array of values with the given types. It returns the
// parsed values or an error when the args could not be parsed.
func parsePositionalArguments(args json.RawMessage, argTypes []reflect.Type) ([]reflect.Value, RPCError) {
	argValues := make([]reflect.Value, len(argTypes))
	params := make([]interface{}, len(argTypes))

	n, err := countArguments(args)
	if err != nil {
		return nil, &invalidParamsError{err.Error()}
	}
	if n != len(argTypes) {
		return nil, &invalidParamsError{fmt.Sprintf("insufficient params, want %d have %d", len(argTypes), n)}
	}

	for i, t := range argTypes {
		if t.Kind() == reflect.Ptr {
			// values must be pointers for the Unmarshal method, reflect.
			// Dereference otherwise reflect.New would create **SomeType
			argValues[i] = reflect.New(t.Elem())
			params[i] = argValues[i].Interface()

			// when not specified blockNumbers are by default latest (-1)
			if blockNumber, ok := params[i].(*BlockNumber); ok {
				*blockNumber = BlockNumber(-1)
			}
		} else {
			argValues[i] = reflect.New(t)
			params[i] = argValues[i].Interface()

			// when not specified blockNumbers are by default latest (-1)
			if blockNumber, ok := params[i].(*BlockNumber); ok {
				*blockNumber = BlockNumber(-1)
			}
		}
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, &invalidParamsError{err.Error()}
	}

	// Convert pointers back to values where necessary
	for i, a := range argValues {
		if a.Kind() != argTypes[i].Kind() {
			argValues[i] = reflect.Indirect(argValues[i])
		}
	}

	return argValues, nil
}

// CreateResponse will create a JSON-RPC success response with the given id and reply as result.
func (c *jsonCodec) CreateResponse(id int64, reply interface{}) interface{} {
	if isHexNum(reflect.TypeOf(reply)) {
		return &jsonSuccessResponse{Version: jsonRPCVersion, Id: id, Result: fmt.Sprintf(`%#x`, reply)}
	}
	return &jsonSuccessResponse{Version: jsonRPCVersion, Id: id, Result: reply}
}

// CreateErrorResponse will create a JSON-RPC error response with the given id and error.
func (c *jsonCodec) CreateErrorResponse(id *int64, err RPCError) interface{} {
	return &jsonErrResponse{Version: jsonRPCVersion, Id: id, Error: jsonError{Code: err.Code(), Message: err.Error()}}
}

// CreateErrorResponseWithInfo will create a JSON-RPC error response with the given id and error.
// info is optional and contains additional information about the error. When an empty string is passed it is ignored.
func (c *jsonCodec) CreateErrorResponseWithInfo(id *int64, err RPCError, info interface{}) interface{} {
	return &jsonErrResponse{Version: jsonRPCVersion, Id: id,
		Error: jsonError{Code: err.Code(), Message: err.Error(), Data: info}}
}

// CreateNotification will create a JSON-RPC notification with the given subscription id and event as params.
func (c *jsonCodec) CreateNotification(subid string, event interface{}) interface{} {
	if isHexNum(reflect.TypeOf(event)) {
		return &jsonNotification{Version: jsonRPCVersion, Method: notificationMethod,
			Params: jsonSubscription{Subscription: subid, Result: fmt.Sprintf(`%#x`, event)}}
	}

	return &jsonNotification{Version: jsonRPCVersion, Method: notificationMethod,
		Params: jsonSubscription{Subscription: subid, Result: event}}
}

// Write message to client
func (c *jsonCodec) Write(res interface{}) error {
	return c.e.Encode(res)
}

// Close the underlying connection
func (c *jsonCodec) Close() {
	if atomic.CompareAndSwapInt32(&c.isClosed, 0, 1) {
		close(c.closed)
		c.rw.Close()
	}
}

// Closed returns a channel which will be closed when Close is called
func (c *jsonCodec) Closed() <-chan interface{} {
	return c.closed
}
