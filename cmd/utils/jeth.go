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

func (self *Jeth) err(call otto.FunctionCall, code int, msg string, id *int64) (response otto.Value) {
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

// UnlockAccount asks the user for the password and than executes the jeth.UnlockAccount callback in the jsre
func (self *Jeth) UnlockAccount(call otto.FunctionCall) (response otto.Value) {
	var cmd, account, passwd string
	timeout := int64(300)
	var ok bool

	if len(call.ArgumentList) == 0 {
		fmt.Println("expected address of account to unlock")
		return otto.FalseValue()
	}

	if len(call.ArgumentList) >= 1 {
		if accountExport, err := call.Argument(0).Export(); err == nil {
			if account, ok = accountExport.(string); ok {
				if len(call.ArgumentList) == 1 {
					fmt.Printf("Unlock account %s\n", account)
					passwd, err = PromptPassword("Passphrase: ", true)
					if err != nil {
						return otto.FalseValue()
					}
				}
			}
		}
	}
	if len(call.ArgumentList) >= 2 {
		if passwdExport, err := call.Argument(1).Export(); err == nil {
			passwd, _ = passwdExport.(string)
		}
	}

	if len(call.ArgumentList) >= 3 {
		if timeoutExport, err := call.Argument(2).Export(); err == nil {
			timeout, _ = timeoutExport.(int64)
		}
	}

	cmd = fmt.Sprintf("jeth.unlockAccount('%s', '%s', %d)", account, passwd, timeout)
	if val, err := call.Otto.Run(cmd); err == nil {
		return val
	}

	return otto.FalseValue()
}

// NewAccount asks the user for the password and than executes the jeth.newAccount callback in the jsre
func (self *Jeth) NewAccount(call otto.FunctionCall) (response otto.Value) {
	if len(call.ArgumentList) == 0 {
		passwd, err := PromptPassword("Passphrase: ", true)
		if err != nil {
			return otto.FalseValue()
		}
		passwd2, err := PromptPassword("Repeat passphrase: ", true)
		if err != nil {
			return otto.FalseValue()
		}

		if passwd != passwd2 {
			fmt.Println("Passphrases don't match")
			return otto.FalseValue()
		}

		cmd := fmt.Sprintf("jeth.newAccount('%s')", passwd)
		if val, err := call.Otto.Run(cmd); err == nil {
			return val
		}
	} else {
		fmt.Println("New account doesn't expect argument(s), you will be prompted for a password")
	}

	return otto.FalseValue()
}

func (self *Jeth) Send(call otto.FunctionCall) (response otto.Value) {
	reqif, err := call.Argument(0).Export()
	if err != nil {
		return self.err(call, -32700, err.Error(), nil)
	}

	jsonreq, err := json.Marshal(reqif)
	var reqs []rpc.JSONRequest
	batch := true
	err = json.Unmarshal(jsonreq, &reqs)
	if err != nil {
		reqs = make([]rpc.JSONRequest, 1)
		err = json.Unmarshal(jsonreq, &reqs[0])
		batch = false
	}

	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		err := self.client.Send(&req)
		if err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}

		result := make(map[string]interface{})
		err = self.client.Recv(&result)
		if err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}

		_, isSuccessResponse := result["result"]
		_, isErrorResponse := result["error"]
		if !isSuccessResponse && !isErrorResponse {
			return self.err(call, -32603, fmt.Sprintf("Invalid response"), new(int64))
		}

		id, _ := result["id"]
		call.Otto.Set("ret_id", id)

		jsonver, _ := result["jsonrpc"]
		call.Otto.Set("ret_jsonrpc", jsonver)

		var payload []byte
		if isSuccessResponse {
			payload, _ = json.Marshal(result["result"])
		} else if isErrorResponse {
			payload, _ = json.Marshal(result["error"])
		}
		call.Otto.Set("ret_result", string(payload))
		call.Otto.Set("response_idx", i)

		response, err = call.Otto.Run(`
		ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
		`)
	}

	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}

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

/*
// handleRequest will handle user agent requests by interacting with the user and sending
// the user response back to the geth service
func (self *Jeth) handleRequest(req *shared.Request) bool {
	var err error
	var args []interface{}
	if err = json.Unmarshal(req.Params, &args); err != nil {
		glog.V(logger.Info).Infof("Unable to parse agent request - %v\n", err)
		return false
	}

	switch req.Method {
	case useragent.AskPasswordMethod:
		return self.askPassword(req.Id, req.Jsonrpc, args)
	case useragent.ConfirmTransactionMethod:
		return self.confirmTransaction(req.Id, req.Jsonrpc, args)
	}

	return false
}

// askPassword will ask the user to supply the password for a given account
func (self *Jeth) askPassword(id interface{}, jsonrpc string, args []interface{}) bool {
	var err error
	var passwd string
	if len(args) >= 1 {
		if account, ok := args[0].(string); ok {
			fmt.Printf("Unlock account %s\n", account)
		} else {
			return false
		}
	}
	passwd, err = PromptPassword("Passphrase: ", true)

	if err = self.client.Send(shared.NewRpcResponse(id, jsonrpc, passwd, err)); err != nil {
		glog.V(logger.Info).Infof("Unable to send user agent ask password response - %v\n", err)
	}

	return err == nil
}

func (self *Jeth) confirmTransaction(id interface{}, jsonrpc string, args []interface{}) bool {
	// Accept all tx which are send from this console
	return self.client.Send(shared.NewRpcResponse(id, jsonrpc, true, nil)) == nil
}
*/

// throwJSExeception panics on an otto value, the Otto VM will then throw msg as a javascript error.
func throwJSExeception(msg interface{}) otto.Value {
	p, _ := otto.ToValue(msg)
	panic(p)
	return p
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
