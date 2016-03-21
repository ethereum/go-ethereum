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

package backends

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// This nil assignment ensures compile time that rpcBackend implements bind.ContractBackend.
var _ bind.ContractBackend = (*rpcBackend)(nil)

// rpcBackend implements bind.ContractBackend, and acts as the data provider to
// Ethereum contracts bound to Go structs. It uses an RPC connection to delegate
// all its functionality.
//
// Note: The current implementation is a blocking one. This should be replaced
// by a proper async version when a real RPC client is created.
type rpcBackend struct {
	client rpc.Client // RPC client connection to interact with an API server
	autoid uint32     // ID number to use for the next API request
	lock   sync.Mutex // Singleton access until we get to request multiplexing
}

// NewRPCBackend creates a new binding backend to an RPC provider that can be
// used to interact with remote contracts.
func NewRPCBackend(client rpc.Client) bind.ContractBackend {
	return &rpcBackend{
		client: client,
	}
}

// request is a JSON RPC request package assembled internally from the client
// method calls.
type request struct {
	JSONRPC string        `json:"jsonrpc"` // Version of the JSON RPC protocol, always set to 2.0
	ID      int           `json:"id"`      // Auto incrementing ID number for this request
	Method  string        `json:"method"`  // Remote procedure name to invoke on the server
	Params  []interface{} `json:"params"`  // List of parameters to pass through (keep types simple)
}

// response is a JSON RPC response package sent back from the API server.
type response struct {
	JSONRPC string          `json:"jsonrpc"` // Version of the JSON RPC protocol, always set to 2.0
	ID      int             `json:"id"`      // Auto incrementing ID number for this request
	Error   json.RawMessage `json:"error"`   // Any error returned by the remote side
	Result  json.RawMessage `json:"result"`  // Whatever the remote side sends us in reply
}

// request forwards an API request to the RPC server, and parses the response.
//
// This is currently painfully non-concurrent, but it will have to do until we
// find the time for niceties like this :P
func (b *rpcBackend) request(method string, params []interface{}) (json.RawMessage, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Ugly hack to serialize an empty list properly
	if params == nil {
		params = []interface{}{}
	}
	// Assemble the request object
	req := &request{
		JSONRPC: "2.0",
		ID:      int(atomic.AddUint32(&b.autoid, 1)),
		Method:  method,
		Params:  params,
	}
	if err := b.client.Send(req); err != nil {
		return nil, err
	}
	res := new(response)
	if err := b.client.Recv(res); err != nil {
		return nil, err
	}
	if len(res.Error) > 0 {
		return nil, fmt.Errorf("remote error: %s", string(res.Error))
	}
	return res.Result, nil
}

// ContractCall implements ContractCaller.ContractCall, delegating the execution of
// a contract call to the remote node, returning the reply to for local processing.
func (b *rpcBackend) ContractCall(contract common.Address, data []byte, pending bool) ([]byte, error) {
	// Pack up the request into an RPC argument
	args := struct {
		To   common.Address `json:"to"`
		Data string         `json:"data"`
	}{
		To:   contract,
		Data: common.ToHex(data),
	}
	// Execute the RPC call and retrieve the response
	block := "latest"
	if pending {
		block = "pending"
	}
	res, err := b.request("eth_call", []interface{}{args, block})
	if err != nil {
		return nil, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return nil, err
	}
	// Convert the response back to a Go byte slice and return
	return common.FromHex(hex), nil
}

// PendingAccountNonce implements ContractTransactor.PendingAccountNonce, delegating
// the current account nonce retrieval to the remote node.
func (b *rpcBackend) PendingAccountNonce(account common.Address) (uint64, error) {
	res, err := b.request("eth_getTransactionCount", []interface{}{account.Hex(), "pending"})
	if err != nil {
		return 0, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return 0, err
	}
	nonce, ok := new(big.Int).SetString(hex, 0)
	if !ok {
		return 0, fmt.Errorf("invalid nonce hex: %s", hex)
	}
	return nonce.Uint64(), nil
}

// SuggestGasPrice implements ContractTransactor.SuggestGasPrice, delegating the
// gas price oracle request to the remote node.
func (b *rpcBackend) SuggestGasPrice() (*big.Int, error) {
	res, err := b.request("eth_gasPrice", nil)
	if err != nil {
		return nil, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return nil, err
	}
	price, ok := new(big.Int).SetString(hex, 0)
	if !ok {
		return nil, fmt.Errorf("invalid price hex: %s", hex)
	}
	return price, nil
}

// EstimateGasLimit implements ContractTransactor.EstimateGasLimit, delegating
// the gas estimation to the remote node.
func (b *rpcBackend) EstimateGasLimit(sender common.Address, contract *common.Address, value *big.Int, data []byte) (*big.Int, error) {
	// Pack up the request into an RPC argument
	args := struct {
		From  common.Address  `json:"from"`
		To    *common.Address `json:"to"`
		Value *rpc.HexNumber  `json:"value"`
		Data  string          `json:"data"`
	}{
		From:  sender,
		To:    contract,
		Data:  common.ToHex(data),
		Value: rpc.NewHexNumber(value),
	}
	// Execute the RPC call and retrieve the response
	res, err := b.request("eth_estimateGas", []interface{}{args})
	if err != nil {
		return nil, err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return nil, err
	}
	estimate, ok := new(big.Int).SetString(hex, 0)
	if !ok {
		return nil, fmt.Errorf("invalid estimate hex: %s", hex)
	}
	return estimate, nil
}

// SendTransaction implements ContractTransactor.SendTransaction, delegating the
// raw transaction injection to the remote node.
func (b *rpcBackend) SendTransaction(tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	res, err := b.request("eth_sendRawTransaction", []interface{}{common.ToHex(data)})
	if err != nil {
		return err
	}
	var hex string
	if err := json.Unmarshal(res, &hex); err != nil {
		return err
	}
	return nil
}
