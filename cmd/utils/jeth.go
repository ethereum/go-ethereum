// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/robertkrimen/otto"
)

type Jeth struct {
	re     *jsre.JSRE
	client rpc.Client
}

// NewJeth create a new backend for the JSRE console
func NewJeth(re *jsre.JSRE, client rpc.Client) *Jeth {
	return &Jeth{re, client}
}

// err returns an error object for the given error code and message.
func (self *Jeth) err(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	m := rpc.JSONErrResponse{
		Version: "2.0",
		Id:      id,
		Error: rpc.JSONError{
			Code:    code,
			Message: msg,
		},
	}

	errObj, _ := json.Marshal(m.Error)
	errRes, _ := json.Marshal(m)

	call.Otto.Run("ret_error = " + string(errObj))
	res, _ := call.Otto.Run("ret_response = " + string(errRes))

	return res
}

// UnlockAccount asks the user for the password and than executes the jeth.UnlockAccount callback in the jsre.
// It will need the public address for the account to unlock as first argument.
// The second argument is an optional string with the password. If not given the user is prompted for the password.
// The third argument is an optional integer which specifies for how long the account will be unlocked (in seconds).
func (self *Jeth) UnlockAccount(call otto.FunctionCall) (response otto.Value) {
	var account, passwd otto.Value
	duration := otto.NullValue()

	if !call.Argument(0).IsString() {
		fmt.Println("first argument must be the account to unlock")
		return otto.FalseValue()
	}

	account = call.Argument(0)

	// if password is not given or as null value -> ask user for password
	if call.Argument(1).IsUndefined() || call.Argument(1).IsNull() {
		fmt.Printf("Unlock account %s\n", account)
		if input, err := Stdin.PasswordPrompt("Passphrase: "); err != nil {
			throwJSExeception(err.Error())
		} else {
			passwd, _ = otto.ToValue(input)
		}
	} else {
		if !call.Argument(1).IsString() {
			throwJSExeception("password must be a string")
		}
		passwd = call.Argument(1)
	}

	// third argument is the duration how long the account must be unlocked.
	// verify that its a number.
	if call.Argument(2).IsDefined() && !call.Argument(2).IsNull() {
		if !call.Argument(2).IsNumber() {
			throwJSExeception("unlock duration must be a number")
		}
		duration = call.Argument(2)
	}

	// jeth.unlockAccount will send the request to the backend.
	if val, err := call.Otto.Call("jeth.unlockAccount", nil, account, passwd, duration); err == nil {
		return val
	} else {
		throwJSExeception(err.Error())
	}

	return otto.FalseValue()
}

// NewAccount asks the user for the password and than executes the jeth.newAccount callback in the jsre
func (self *Jeth) NewAccount(call otto.FunctionCall) (response otto.Value) {
	var passwd string
	if len(call.ArgumentList) == 0 {
		var err error
		passwd, err = Stdin.PasswordPrompt("Passphrase: ")
		if err != nil {
			return otto.FalseValue()
		}
		passwd2, err := Stdin.PasswordPrompt("Repeat passphrase: ")
		if err != nil {
			return otto.FalseValue()
		}

		if passwd != passwd2 {
			fmt.Println("Passphrases don't match")
			return otto.FalseValue()
		}
	} else if len(call.ArgumentList) == 1 && call.Argument(0).IsString() {
		passwd, _ = call.Argument(0).ToString()
	} else {
		fmt.Println("expected 0 or 1 string argument")
		return otto.FalseValue()
	}

	ret, err := call.Otto.Call("jeth.newAccount", nil, passwd)
	if err == nil {
		return ret
	}
	fmt.Println(err)
	return otto.FalseValue()
}

// Send will serialize the first argument, send it to the node and returns the response.
func (self *Jeth) Send(call otto.FunctionCall) (response otto.Value) {
	// verify we got a batch request (array) or a single request (object)
	ro := call.Argument(0).Object()
	if ro == nil || (ro.Class() != "Array" && ro.Class() != "Object") {
		throwJSExeception("Internal Error: request must be an object or array")
	}

	// convert otto vm arguments to go values by JSON serialising and parsing.
	data, err := call.Otto.Call("JSON.stringify", nil, ro)
	if err != nil {
		throwJSExeception(err.Error())
	}

	jsonreq, _ := data.ToString()

	// parse arguments to JSON rpc requests, either to an array (batch) or to a single request.
	var reqs []rpc.JSONRequest
	batch := true
	if err = json.Unmarshal([]byte(jsonreq), &reqs); err != nil {
		// single request?
		reqs = make([]rpc.JSONRequest, 1)
		if err = json.Unmarshal([]byte(jsonreq), &reqs[0]); err != nil {
			throwJSExeception("invalid request")
		}
		batch = false
	}

	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		if err := self.client.Send(&req); err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}

		result := make(map[string]interface{})
		if err = self.client.Recv(&result); err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}

		id, _ := result["id"]
		jsonver, _ := result["jsonrpc"]

		call.Otto.Set("ret_id", id)
		call.Otto.Set("ret_jsonrpc", jsonver)
		call.Otto.Set("response_idx", i)

		// call was successful
		if res, ok := result["result"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
			`)
			continue
		}

		// request returned an error
		if res, ok := result["error"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, error: JSON.parse(ret_result) };
			`)
			continue
		}

		return self.err(call, -32603, fmt.Sprintf("Invalid response"), new(int64))
	}

	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}

	// if a callback was given execute it.
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

// throwJSExeception panics on an otto value, the Otto VM will then throw msg as a javascript error.
func throwJSExeception(msg interface{}) otto.Value {
	p, _ := otto.ToValue(msg)
	panic(p)
}

// Sleep will halt the console for arg[0] seconds.
func (self *Jeth) Sleep(call otto.FunctionCall) (response otto.Value) {
	if len(call.ArgumentList) >= 1 {
		if call.Argument(0).IsNumber() {
			sleep, _ := call.Argument(0).ToInteger()
			time.Sleep(time.Duration(sleep) * time.Second)
			return otto.TrueValue()
		}
	}
	return throwJSExeception("usage: sleep(<sleep in seconds>)")
}

// SleepBlocks will wait for a specified number of new blocks or max for a
// given of seconds. sleepBlocks(nBlocks[, maxSleep]).
func (self *Jeth) SleepBlocks(call otto.FunctionCall) (response otto.Value) {
	nBlocks := int64(0)
	maxSleep := int64(9999999999999999) // indefinitely

	nArgs := len(call.ArgumentList)

	if nArgs == 0 {
		throwJSExeception("usage: sleepBlocks(<n blocks>[, max sleep in seconds])")
	}

	if nArgs >= 1 {
		if call.Argument(0).IsNumber() {
			nBlocks, _ = call.Argument(0).ToInteger()
		} else {
			throwJSExeception("expected number as first argument")
		}
	}

	if nArgs >= 2 {
		if call.Argument(1).IsNumber() {
			maxSleep, _ = call.Argument(1).ToInteger()
		} else {
			throwJSExeception("expected number as second argument")
		}
	}

	// go through the console, this will allow web3 to call the appropriate
	// callbacks if a delayed response or notification is received.
	currentBlockNr := func() int64 {
		result, err := call.Otto.Run("eth.blockNumber")
		if err != nil {
			throwJSExeception(err.Error())
		}
		blockNr, err := result.ToInteger()
		if err != nil {
			throwJSExeception(err.Error())
		}
		return blockNr
	}

	targetBlockNr := currentBlockNr() + nBlocks
	deadline := time.Now().Add(time.Duration(maxSleep) * time.Second)

	for time.Now().Before(deadline) {
		if currentBlockNr() >= targetBlockNr {
			return otto.TrueValue()
		}
		time.Sleep(time.Second)
	}

	return otto.FalseValue()
}
