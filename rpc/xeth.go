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
