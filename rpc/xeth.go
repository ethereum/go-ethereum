// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

// Package rpc implements the Ethereum JSON-RPC API.
package rpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

// Xeth is a native API interface to a remote node.
type Xeth struct {
	client comms.EthereumClient
	reqId  uint32
}

// NewXeth constructs a new native API interface to a remote node.
func NewXeth(client comms.EthereumClient) *Xeth {
	return &Xeth{
		client: client,
	}
}

// Call invokes a method with the given parameters are the remote node.
func (self *Xeth) Call(method string, params []interface{}) (map[string]interface{}, error) {
	// Assemble the json RPC request
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req := &shared.Request{
		Id:      atomic.AddUint32(&self.reqId, 1),
		Jsonrpc: "2.0",
		Method:  method,
		Params:  data,
	}
	// Send the request over and process the response
	if err := self.client.Send(req); err != nil {
		return nil, err
	}
	res, err := self.client.Recv()
	if err != nil {
		return nil, err
	}
	value, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Invalid response type: have %v, want %v", reflect.TypeOf(res), reflect.TypeOf(make(map[string]interface{})))
	}
	return value, nil
}
