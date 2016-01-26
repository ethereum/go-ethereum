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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"
)

type Service struct{}

type Args struct {
	S string
}

func (s *Service) NoArgsRets() {
}

type Result struct {
	String string
	Int    int
	Args   *Args
}

func (s *Service) Echo(str string, i int, args *Args) Result {
	return Result{str, i, args}
}

func (s *Service) EchoWithCtx(ctx context.Context, str string, i int, args *Args) Result {
	return Result{str, i, args}
}

func (s *Service) Rets() (string, error) {
	return "", nil
}

func (s *Service) InvalidRets1() (error, string) {
	return nil, ""
}

func (s *Service) InvalidRets2() (string, string) {
	return "", ""
}

func (s *Service) InvalidRets3() (string, string, error) {
	return "", "", nil
}

func (s *Service) Subscription() (Subscription, error) {
	return NewSubscription(nil), nil
}

func TestServerRegisterName(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("calc", service); err != nil {
		t.Fatalf("%v", err)
	}

	if len(server.services) != 2 {
		t.Fatalf("Expected 2 service entries, got %d", len(server.services))
	}

	svc, ok := server.services["calc"]
	if !ok {
		t.Fatalf("Expected service calc to be registered")
	}

	if len(svc.callbacks) != 4 {
		t.Errorf("Expected 4 callbacks for service 'calc', got %d", len(svc.callbacks))
	}

	if len(svc.subscriptions) != 1 {
		t.Errorf("Expected 1 subscription for service 'calc', got %d", len(svc.subscriptions))
	}
}

// dummy codec used for testing RPC method execution
type ServerTestCodec struct {
	counter int
	input   []byte
	output  string
	closer  chan interface{}
}

func (c *ServerTestCodec) ReadRequestHeaders() ([]rpcRequest, bool, RPCError) {
	c.counter += 1

	if c.counter == 1 {
		var req JSONRequest
		json.Unmarshal(c.input, &req)
		return []rpcRequest{rpcRequest{id: *req.Id, isPubSub: false, service: "test", method: req.Method, params: req.Payload}}, false, nil
	}

	// requests are executes in parallel, wait a bit before returning an error so that the previous request has time to
	// be executed
	timer := time.NewTimer(time.Duration(2) * time.Second)
	<-timer.C

	return nil, false, &invalidRequestError{"connection closed"}
}

func (c *ServerTestCodec) ParseRequestArguments(argTypes []reflect.Type, payload interface{}) ([]reflect.Value, RPCError) {

	args, _ := payload.(json.RawMessage)

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

func (c *ServerTestCodec) CreateResponse(id int64, reply interface{}) interface{} {
	return &JSONSuccessResponse{Version: jsonRPCVersion, Id: id, Result: reply}
}

func (c *ServerTestCodec) CreateErrorResponse(id *int64, err RPCError) interface{} {
	return &JSONErrResponse{Version: jsonRPCVersion, Id: id, Error: JSONError{Code: err.Code(), Message: err.Error()}}
}

func (c *ServerTestCodec) CreateErrorResponseWithInfo(id *int64, err RPCError, info interface{}) interface{} {
	return &JSONErrResponse{Version: jsonRPCVersion, Id: id,
		Error: JSONError{Code: err.Code(), Message: err.Error(), Data: info}}
}

func (c *ServerTestCodec) CreateNotification(subid string, event interface{}) interface{} {
	return &jsonNotification{Version: jsonRPCVersion, Method: notificationMethod,
		Params: jsonSubscription{Subscription: subid, Result: event}}
}

func (c *ServerTestCodec) Write(msg interface{}) error {
	if len(c.output) == 0 { // only capture first response
		if o, err := json.Marshal(msg); err != nil {
			return err
		} else {
			c.output = string(o)
		}
	}

	return nil
}

func (c *ServerTestCodec) Close() {
	close(c.closer)
}

func (c *ServerTestCodec) Closed() <-chan interface{} {
	return c.closer
}

func TestServerMethodExecution(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("test", service); err != nil {
		t.Fatalf("%v", err)
	}

	id := int64(12345)
	req := JSONRequest{
		Method:  "echo",
		Version: "2.0",
		Id:      &id,
	}
	args := []interface{}{"string arg", 1122, &Args{"qwerty"}}
	req.Payload, _ = json.Marshal(&args)

	input, _ := json.Marshal(&req)
	codec := &ServerTestCodec{input: input, closer: make(chan interface{})}
	go server.ServeCodec(codec)

	<-codec.closer

	expected := `{"jsonrpc":"2.0","id":12345,"result":{"String":"string arg","Int":1122,"Args":{"S":"qwerty"}}}`

	if expected != codec.output {
		t.Fatalf("expected %s, got %s\n", expected, codec.output)
	}
}

func TestServerMethodWithCtx(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("test", service); err != nil {
		t.Fatalf("%v", err)
	}

	id := int64(12345)
	req := JSONRequest{
		Method:  "echoWithCtx",
		Version: "2.0",
		Id:      &id,
	}
	args := []interface{}{"string arg", 1122, &Args{"qwerty"}}
	req.Payload, _ = json.Marshal(&args)

	input, _ := json.Marshal(&req)
	codec := &ServerTestCodec{input: input, closer: make(chan interface{})}
	go server.ServeCodec(codec)

	<-codec.closer

	expected := `{"jsonrpc":"2.0","id":12345,"result":{"String":"string arg","Int":1122,"Args":{"S":"qwerty"}}}`

	if expected != codec.output {
		t.Fatalf("expected %s, got %s\n", expected, codec.output)
	}
}
