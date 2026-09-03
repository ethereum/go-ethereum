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

package console

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/internal/jsre"
	"github.com/ethereum/go-ethereum/rpc"
)

// bridge is a collection of JavaScript utility methods to bride the .js runtime
// environment and the Go RPC connection backing the remote method calls.
type bridge struct {
	client   *rpc.Client         // RPC client to execute Ethereum requests through
	prompter prompt.UserPrompter // Input prompter to allow interactive user feedback
	printer  io.Writer           // Output writer to serialize any display strings to
}

// newBridge creates a new JavaScript wrapper around an RPC client.
func newBridge(client *rpc.Client, prompter prompt.UserPrompter, printer io.Writer) *bridge {
	return &bridge{
		client:   client,
		prompter: prompter,
		printer:  printer,
	}
}

// Sleep will block the console for the specified number of seconds.
func (b *bridge) Sleep(call jsre.Call) (goja.Value, error) {
	if nArgs := len(call.Arguments); nArgs < 1 {
		return nil, errors.New("usage: sleep(<number of seconds>)")
	}
	sleepObj := call.Argument(0)
	if goja.IsUndefined(sleepObj) || goja.IsNull(sleepObj) || !isNumber(sleepObj) {
		return nil, errors.New("usage: sleep(<number of seconds>)")
	}
	sleep := sleepObj.ToFloat()
	time.Sleep(time.Duration(sleep * float64(time.Second)))
	return call.VM.ToValue(true), nil
}

// SleepBlocks will block the console for a specified number of new blocks optionally
// until the given timeout is reached.
func (b *bridge) SleepBlocks(call jsre.Call) (goja.Value, error) {
	// Parse the input parameters for the sleep.
	var (
		blocks = int64(0)
		sleep  = int64(9999999999999999) // indefinitely
	)
	nArgs := len(call.Arguments)
	if nArgs == 0 {
		return nil, errors.New("usage: sleepBlocks(<n blocks>[, max sleep in seconds])")
	}
	if nArgs >= 1 {
		if goja.IsNull(call.Argument(0)) || goja.IsUndefined(call.Argument(0)) || !isNumber(call.Argument(0)) {
			return nil, errors.New("expected number as first argument")
		}
		blocks = call.Argument(0).ToInteger()
	}
	if nArgs >= 2 {
		if goja.IsNull(call.Argument(1)) || goja.IsUndefined(call.Argument(1)) || !isNumber(call.Argument(1)) {
			return nil, errors.New("expected number as second argument")
		}
		sleep = call.Argument(1).ToInteger()
	}

	// Poll the current block number until either it or a timeout is reached.
	deadline := time.Now().Add(time.Duration(sleep) * time.Second)
	var lastNumber hexutil.Uint64
	if err := b.client.Call(&lastNumber, "eth_blockNumber"); err != nil {
		return nil, err
	}
	for time.Now().Before(deadline) {
		var number hexutil.Uint64
		if err := b.client.Call(&number, "eth_blockNumber"); err != nil {
			return nil, err
		}
		if number != lastNumber {
			lastNumber = number
			blocks--
		}
		if blocks <= 0 {
			break
		}
		time.Sleep(time.Second)
	}
	return call.VM.ToValue(true), nil
}

type jsonrpcCall struct {
	ID     int64
	Method string
	Params []interface{}
}

// Send implements the web3 provider "send" method.
func (b *bridge) Send(call jsre.Call) (goja.Value, error) {
	// Remarshal the request into a Go value.
	reqVal, err := call.Argument(0).ToObject(call.VM).MarshalJSON()
	if err != nil {
		return nil, err
	}

	var (
		rawReq = string(reqVal)
		dec    = json.NewDecoder(strings.NewReader(rawReq))
		reqs   []jsonrpcCall
		batch  bool
	)
	dec.UseNumber() // avoid float64s
	if rawReq[0] == '[' {
		batch = true
		dec.Decode(&reqs)
	} else {
		batch = false
		reqs = make([]jsonrpcCall, 1)
		dec.Decode(&reqs[0])
	}

	// Execute the requests.
	var resps []*goja.Object
	for _, req := range reqs {
		resp := call.VM.NewObject()
		resp.Set("jsonrpc", "2.0")
		resp.Set("id", req.ID)

		var result json.RawMessage
		if err = b.client.Call(&result, req.Method, req.Params...); err == nil {
			if result == nil {
				// Special case null because it is decoded as an empty
				// raw message for some reason.
				resp.Set("result", goja.Null())
			} else {
				JSON := call.VM.Get("JSON").ToObject(call.VM)
				parse, callable := goja.AssertFunction(JSON.Get("parse"))
				if !callable {
					return nil, errors.New("JSON.parse is not a function")
				}
				resultVal, err := parse(goja.Null(), call.VM.ToValue(string(result)))
				if err != nil {
					setError(resp, -32603, err.Error(), nil)
				} else {
					resp.Set("result", resultVal)
				}
			}
		} else {
			code := -32603
			var data interface{}
			if err, ok := err.(rpc.Error); ok {
				code = err.ErrorCode()
			}
			if err, ok := err.(rpc.DataError); ok {
				data = err.ErrorData()
			}
			setError(resp, code, err.Error(), data)
		}
		resps = append(resps, resp)
	}
	// Return the responses either to the callback (if supplied)
	// or directly as the return value.
	var result goja.Value
	if batch {
		result = call.VM.ToValue(resps)
	} else {
		result = resps[0]
	}
	if fn, isFunc := goja.AssertFunction(call.Argument(1)); isFunc {
		fn(goja.Null(), goja.Null(), result)
		return goja.Undefined(), nil
	}
	return result, nil
}

func setError(resp *goja.Object, code int, msg string, data interface{}) {
	err := make(map[string]interface{})
	err["code"] = code
	err["message"] = msg
	if data != nil {
		err["data"] = data
	}
	resp.Set("error", err)
}

// isNumber returns true if input value is a JS number.
func isNumber(v goja.Value) bool {
	k := v.ExportType().Kind()
	return k >= reflect.Int && k <= reflect.Float64
}

func getObject(vm *goja.Runtime, name string) *goja.Object {
	v := vm.Get(name)
	if v == nil {
		return nil
	}
	return v.ToObject(vm)
}
