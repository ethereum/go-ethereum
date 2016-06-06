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

package console

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
)

// bridge is a collection of JavaScript utility methods to bride the .js runtime
// environment and the Go RPC connection backing the remote method calls.
type bridge struct {
	client   rpc.Client   // RPC client to execute Ethereum requests through
	prompter UserPrompter // Input prompter to allow interactive user feedback
	printer  io.Writer    // Output writer to serialize any display strings to
}

// newBridge creates a new JavaScript wrapper around an RPC client.
func newBridge(client rpc.Client, prompter UserPrompter, printer io.Writer) *bridge {
	return &bridge{
		client:   client,
		prompter: prompter,
		printer:  printer,
	}
}

// NewAccount is a wrapper around the personal.newAccount RPC method that uses a
// non-echoing password prompt to aquire the passphrase and executes the original
// RPC method (saved in jeth.newAccount) with it to actually execute the RPC call.
func (b *bridge) NewAccount(call otto.FunctionCall) (response otto.Value) {
	var (
		password string
		confirm  string
		err      error
	)
	switch {
	// No password was specified, prompt the user for it
	case len(call.ArgumentList) == 0:
		if password, err = b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(err.Error())
		}
		if confirm, err = b.prompter.PromptPassword("Repeat passphrase: "); err != nil {
			throwJSException(err.Error())
		}
		if password != confirm {
			throwJSException("passphrases don't match!")
		}

	// A single string password was specified, use that
	case len(call.ArgumentList) == 1 && call.Argument(0).IsString():
		password, _ = call.Argument(0).ToString()

	// Otherwise fail with some error
	default:
		throwJSException("expected 0 or 1 string argument")
	}
	// Password aquired, execute the call and return
	ret, err := call.Otto.Call("jeth.newAccount", nil, password)
	if err != nil {
		throwJSException(err.Error())
	}
	return ret
}

// UnlockAccount is a wrapper around the personal.unlockAccount RPC method that
// uses a non-echoing password prompt to aquire the passphrase and executes the
// original RPC method (saved in jeth.unlockAccount) with it to actually execute
// the RPC call.
func (b *bridge) UnlockAccount(call otto.FunctionCall) (response otto.Value) {
	// Make sure we have an account specified to unlock
	if !call.Argument(0).IsString() {
		throwJSException("first argument must be the account to unlock")
	}
	account := call.Argument(0)

	// If password is not given or is the null value, prompt the user for it
	var passwd otto.Value

	if call.Argument(1).IsUndefined() || call.Argument(1).IsNull() {
		fmt.Fprintf(b.printer, "Unlock account %s\n", account)
		if input, err := b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(err.Error())
		} else {
			passwd, _ = otto.ToValue(input)
		}
	} else {
		if !call.Argument(1).IsString() {
			throwJSException("password must be a string")
		}
		passwd = call.Argument(1)
	}
	// Third argument is the duration how long the account must be unlocked.
	duration := otto.NullValue()
	if call.Argument(2).IsDefined() && !call.Argument(2).IsNull() {
		if !call.Argument(2).IsNumber() {
			throwJSException("unlock duration must be a number")
		}
		duration = call.Argument(2)
	}
	// Send the request to the backend and return
	val, err := call.Otto.Call("jeth.unlockAccount", nil, account, passwd, duration)
	if err != nil {
		throwJSException(err.Error())
	}
	return val
}

// Sleep will block the console for the specified number of seconds.
func (b *bridge) Sleep(call otto.FunctionCall) (response otto.Value) {
	if call.Argument(0).IsNumber() {
		sleep, _ := call.Argument(0).ToInteger()
		time.Sleep(time.Duration(sleep) * time.Second)
		return otto.TrueValue()
	}
	return throwJSException("usage: sleep(<number of seconds>)")
}

// SleepBlocks will block the console for a specified number of new blocks optionally
// until the given timeout is reached.
func (b *bridge) SleepBlocks(call otto.FunctionCall) (response otto.Value) {
	var (
		blocks = int64(0)
		sleep  = int64(9999999999999999) // indefinitely
	)
	// Parse the input parameters for the sleep
	nArgs := len(call.ArgumentList)
	if nArgs == 0 {
		throwJSException("usage: sleepBlocks(<n blocks>[, max sleep in seconds])")
	}
	if nArgs >= 1 {
		if call.Argument(0).IsNumber() {
			blocks, _ = call.Argument(0).ToInteger()
		} else {
			throwJSException("expected number as first argument")
		}
	}
	if nArgs >= 2 {
		if call.Argument(1).IsNumber() {
			sleep, _ = call.Argument(1).ToInteger()
		} else {
			throwJSException("expected number as second argument")
		}
	}
	// go through the console, this will allow web3 to call the appropriate
	// callbacks if a delayed response or notification is received.
	blockNumber := func() int64 {
		result, err := call.Otto.Run("eth.blockNumber")
		if err != nil {
			throwJSException(err.Error())
		}
		block, err := result.ToInteger()
		if err != nil {
			throwJSException(err.Error())
		}
		return block
	}
	// Poll the current block number until either it ot a timeout is reached
	targetBlockNr := blockNumber() + blocks
	deadline := time.Now().Add(time.Duration(sleep) * time.Second)

	for time.Now().Before(deadline) {
		if blockNumber() >= targetBlockNr {
			return otto.TrueValue()
		}
		time.Sleep(time.Second)
	}
	return otto.FalseValue()
}

// Send will serialize the first argument, send it to the node and returns the response.
func (b *bridge) Send(call otto.FunctionCall) (response otto.Value) {
	// Ensure that we've got a batch request (array) or a single request (object)
	arg := call.Argument(0).Object()
	if arg == nil || (arg.Class() != "Array" && arg.Class() != "Object") {
		throwJSException("request must be an object or array")
	}
	// Convert the otto VM arguments to Go values
	data, err := call.Otto.Call("JSON.stringify", nil, arg)
	if err != nil {
		throwJSException(err.Error())
	}
	reqjson, err := data.ToString()
	if err != nil {
		throwJSException(err.Error())
	}

	var (
		reqs  []rpc.JSONRequest
		batch = true
	)
	if err = json.Unmarshal([]byte(reqjson), &reqs); err != nil {
		// single request?
		reqs = make([]rpc.JSONRequest, 1)
		if err = json.Unmarshal([]byte(reqjson), &reqs[0]); err != nil {
			throwJSException("invalid request")
		}
		batch = false
	}
	// Iteratively execute the requests
	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		// Execute the RPC request and parse the reply
		if err = b.client.Send(&req); err != nil {
			return newErrorResponse(call, -32603, err.Error(), req.Id)
		}
		result := make(map[string]interface{})
		if err = b.client.Recv(&result); err != nil {
			return newErrorResponse(call, -32603, err.Error(), req.Id)
		}
		// Feed the reply back into the JavaScript runtime environment
		id, _ := result["id"]
		jsonver, _ := result["jsonrpc"]

		call.Otto.Set("ret_id", id)
		call.Otto.Set("ret_jsonrpc", jsonver)
		call.Otto.Set("response_idx", i)

		if res, ok := result["result"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
			`)
			continue
		}
		if res, ok := result["error"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, error: JSON.parse(ret_result) };
			`)
			continue
		}
		return newErrorResponse(call, -32603, fmt.Sprintf("Invalid response"), new(int64))
	}
	// Convert single requests back from batch ones
	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}
	// Execute any registered callbacks
	if call.Argument(1).IsObject() {
		call.Otto.Set("callback", call.Argument(1))
		call.Otto.Run(`
		if (Object.prototype.toString.call(callback) == '[object Function]') {
			callback(null, ret_response);
		}
		`)
	}
	return
}

// throwJSException panics on an otto.Value. The Otto VM will recover from the
// Go panic and throw msg as a JavaScript error.
func throwJSException(msg interface{}) otto.Value {
	val, err := otto.ToValue(msg)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JavaScript exception %v: %v", msg, err)
	}
	panic(val)
}

// newErrorResponse creates a JSON RPC error response for a specific request id,
// containing the specified error code and error message. Beside returning the
// error to the caller, it also sets the ret_error and ret_response JavaScript
// variables.
func newErrorResponse(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	// Bundle the error into a JSON RPC call response
	res := rpc.JSONErrResponse{
		Version: rpc.JSONRPCVersion,
		Id:      id,
		Error: rpc.JSONError{
			Code:    code,
			Message: msg,
		},
	}
	// Serialize the error response into JavaScript variables
	errObj, err := json.Marshal(res.Error)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JSON RPC error: %v", err)
	}
	resObj, err := json.Marshal(res)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JSON RPC error response: %v", err)
	}

	if _, err = call.Otto.Run("ret_error = " + string(errObj)); err != nil {
		glog.V(logger.Error).Infof("Failed to set `ret_error` to the occurred error: %v", err)
	}
	resVal, err := call.Otto.Run("ret_response = " + string(resObj))
	if err != nil {
		glog.V(logger.Error).Infof("Failed to set `ret_response` to the JSON RPC response: %v", err)
	}
	return resVal
}
