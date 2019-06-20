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
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/rpc"
)

// bridge is a collection of JavaScript utility methods to bride the .js runtime
// environment and the Go RPC connection backing the remote method calls.
type bridge struct {
	client   *rpc.Client   // RPC client to execute Ethereum requests through
	prompter UserPrompter  // Input prompter to allow interactive user feedback
	printer  io.Writer     // Output writer to serialize any display strings to
	runtime  *goja.Runtime // Pointer to the JS runtime
}

// newBridge creates a new JavaScript wrapper around an RPC client.
func newBridge(client *rpc.Client, prompter UserPrompter, printer io.Writer, runtime *goja.Runtime) *bridge {
	return &bridge{
		client:   client,
		prompter: prompter,
		printer:  printer,
		runtime:  runtime,
	}
}

func IsNumber(v goja.Value) bool {
	switch v.ExportType().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// NewAccount is a wrapper around the personal.newAccount RPC method that uses a
// non-echoing password prompt to acquire the passphrase and executes the original
// RPC method (saved in jeth.newAccount) with it to actually execute the RPC call.
func (b *bridge) NewAccount(call goja.FunctionCall) (response goja.Value) {
	var (
		password string
		confirm  string
		err      error
	)
	switch {
	// No password was specified, prompt the user for it
	case len(call.Arguments) == 0:
		if password, err = b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(b.runtime, err.Error())
		}
		if confirm, err = b.prompter.PromptPassword("Repeat passphrase: "); err != nil {
			throwJSException(b.runtime, err.Error())
		}
		if password != confirm {
			throwJSException(b.runtime, "passphrases don't match!")
		}

	// A single string password was specified, use that
	case len(call.Arguments) == 1 && call.Argument(0).ToString() != nil:
		password = call.Argument(0).ToString().String()

	// Otherwise fail with some error
	default:
		throwJSException(b.runtime, "expected 0 or 1 string argument")
	}
	// Password acquired, execute the call and return
	newAccount, callable := goja.AssertFunction(b.runtime.Get("jeth.newAccount"))
	if !callable {
		panic(b.runtime.ToValue("jeth.newAccount isn't callable"))
	}
	ret, err := newAccount(goja.Null(), b.runtime.ToValue(password))
	if err != nil {
		panic(b.runtime.ToValue(err.Error()))
	}
	return ret
}

// OpenWallet is a wrapper around personal.openWallet which can interpret and
// react to certain error messages, such as the Trezor PIN matrix request.
func (b *bridge) OpenWallet(call goja.FunctionCall) (response goja.Value) {
	// Make sure we have a wallet specified to open
	if call.Argument(0).ToObject(b.runtime).ClassName() != "String" {
		panic(b.runtime.ToValue("first argument must be the wallet URL to open"))
	}
	wallet := call.Argument(0)

	var passwd goja.Value
	if goja.IsUndefined(call.Argument(1)) || goja.IsNull(call.Argument(1)) {
		passwd = b.runtime.ToValue("")
	} else {
		passwd = call.Argument(1)
	}
	// Open the wallet and return if successful in itself
	openWallet, callable := goja.AssertFunction(b.runtime.Get("jeth.openWallet"))
	if !callable {
		panic(b.runtime.ToValue("jeth.openWallet is not callable"))
	}
	val, err := openWallet(goja.Null(), wallet, passwd)
	if err == nil {
		return val
	}

	// Wallet open failed, report error unless it's a PIN or PUK entry
	switch {
	case strings.HasSuffix(err.Error(), usbwallet.ErrTrezorPINNeeded.Error()):
		val, err = b.readPinAndReopenWallet(call)
		if err == nil {
			return val
		}
		val, err = b.readPassphraseAndReopenWallet(call)
		if err != nil {
			throwJSException(b.runtime, err.Error())
		}

	case strings.HasSuffix(err.Error(), scwallet.ErrPairingPasswordNeeded.Error()):
		// PUK input requested, fetch from the user and call open again
		if input, err := b.prompter.PromptPassword("Please enter the pairing password: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			passwd = b.runtime.ToValue(input)
		}
		if val, err = openWallet(goja.Null(), wallet, passwd); err != nil {
			if !strings.HasSuffix(err.Error(), scwallet.ErrPINNeeded.Error()) {
				throwJSException(b.runtime, err.Error())
			} else {
				// PIN input requested, fetch from the user and call open again
				if input, err := b.prompter.PromptPassword("Please enter current PIN: "); err != nil {
					throwJSException(b.runtime, err.Error())
				} else {
					passwd = b.runtime.ToValue(input)
				}
				if val, err = openWallet(goja.Null(), wallet, passwd); err != nil {
					throwJSException(b.runtime, err.Error())
				}
			}
		}

	case strings.HasSuffix(err.Error(), scwallet.ErrPINUnblockNeeded.Error()):
		// PIN unblock requested, fetch PUK and new PIN from the user
		var pukpin string
		if input, err := b.prompter.PromptPassword("Please enter current PUK: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			pukpin = input
		}
		if input, err := b.prompter.PromptPassword("Please enter new PIN: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			pukpin += input
		}
		passwd = b.runtime.ToValue(pukpin)
		if val, err = openWallet(goja.Null(), wallet, passwd); err != nil {
			throwJSException(b.runtime, err.Error())
		}

	case strings.HasSuffix(err.Error(), scwallet.ErrPINNeeded.Error()):
		// PIN input requested, fetch from the user and call open again
		if input, err := b.prompter.PromptPassword("Please enter current PIN: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			passwd = b.runtime.ToValue(input)
		}
		if val, err = openWallet(goja.Null(), wallet, passwd); err != nil {
			throwJSException(b.runtime, err.Error())
		}

	default:
		// Unknown error occurred, drop to the user
		throwJSException(b.runtime, err.Error())
	}
	return val
}

func (b *bridge) readPassphraseAndReopenWallet(call goja.FunctionCall) (goja.Value, error) {
	var passwd goja.Value
	wallet := call.Argument(0)
	if input, err := b.prompter.PromptPassword("Please enter your passphrase: "); err != nil {
		throwJSException(b.runtime, err.Error())
	} else {
		passwd = b.runtime.ToValue(input)
	}
	openWallet, callable := goja.AssertFunction(b.runtime.Get("jeth.openWallet"))
	if !callable {
		return nil, fmt.Errorf("jeth.openWallet is not callable")
	}
	return openWallet(goja.Null(), wallet, passwd)
}

func (b *bridge) readPinAndReopenWallet(call goja.FunctionCall) (goja.Value, error) {
	var passwd goja.Value
	wallet := call.Argument(0)
	// Trezor PIN matrix input requested, display the matrix to the user and fetch the data
	fmt.Fprintf(b.printer, "Look at the device for number positions\n\n")
	fmt.Fprintf(b.printer, "7 | 8 | 9\n")
	fmt.Fprintf(b.printer, "--+---+--\n")
	fmt.Fprintf(b.printer, "4 | 5 | 6\n")
	fmt.Fprintf(b.printer, "--+---+--\n")
	fmt.Fprintf(b.printer, "1 | 2 | 3\n\n")

	if input, err := b.prompter.PromptPassword("Please enter current PIN: "); err != nil {
		throwJSException(b.runtime, err.Error())
	} else {
		passwd = b.runtime.ToValue(input)
	}
	openWallet, callable := goja.AssertFunction(b.runtime.Get("jeth.openWallet"))
	if !callable {
		return nil, fmt.Errorf("jeth.openWallet is not callable")
	}
	return openWallet(goja.Null(), wallet, passwd)
}

// UnlockAccount is a wrapper around the personal.unlockAccount RPC method that
// uses a non-echoing password prompt to acquire the passphrase and executes the
// original RPC method (saved in jeth.unlockAccount) with it to actually execute
// the RPC call.
func (b *bridge) UnlockAccount(call goja.FunctionCall) (response goja.Value) {
	// Make sure we have an account specified to unlock
	if call.Argument(0).ExportType().Kind() != reflect.String {
		throwJSException(b.runtime, "first argument must be the account to unlock")
	}
	account := call.Argument(0)

	// If password is not given or is the null value, prompt the user for it
	var passwd goja.Value

	if goja.IsUndefined(call.Argument(1)) || goja.IsNull(call.Argument(1)) {
		fmt.Fprintf(b.printer, "Unlock account %s\n", account)
		if input, err := b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			passwd = b.runtime.ToValue(input)
		}
	} else {
		if call.Argument(1).ExportType().Kind() != reflect.String {
			throwJSException(b.runtime, "password must be a string")
		}
		passwd = call.Argument(1)
	}
	// Third argument is the duration how long the account must be unlocked.
	duration := goja.Null()
	if !goja.IsUndefined(call.Argument(2)) && !goja.IsNull(call.Argument(2)) {
		if !IsNumber(call.Argument(2)) {
			throwJSException(b.runtime, "unlock duration must be a number")
		}
		duration = call.Argument(2)
	}
	// Send the request to the backend and return
	unlockAccount, callable := goja.AssertFunction(b.runtime.Get("jeth.unlockAccount"))
	if !callable {
		throwJSException(b.runtime, "jeth.unlockAccount is not callable")
	}
	val, err := unlockAccount(goja.Null(), account, passwd, duration)
	if err != nil {
		throwJSException(b.runtime, err.Error())
	}
	return val
}

// Sign is a wrapper around the personal.sign RPC method that uses a non-echoing password
// prompt to acquire the passphrase and executes the original RPC method (saved in
// jeth.sign) with it to actually execute the RPC call.
func (b *bridge) Sign(call goja.FunctionCall) (response goja.Value) {
	var (
		message = call.Argument(0)
		account = call.Argument(1)
		passwd  = call.Argument(2)
	)

	if message.ExportType().Kind() != reflect.String {
		throwJSException(b.runtime, "first argument must be the message to sign")
	}
	if account.ExportType().Kind() != reflect.String {
		throwJSException(b.runtime, "second argument must be the account to sign with")
	}

	// if the password is not given or null ask the user and ensure password is a string
	if goja.IsUndefined(passwd) || goja.IsNull(passwd) {
		fmt.Fprintf(b.printer, "Give password for account %s\n", account)
		if input, err := b.prompter.PromptPassword("Passphrase: "); err != nil {
			throwJSException(b.runtime, err.Error())
		} else {
			passwd = b.runtime.ToValue(input)
		}
	}
	if passwd.ExportType().Kind() != reflect.String {
		throwJSException(b.runtime, "third argument must be the password to unlock the account")
	}

	// Send the request to the backend and return
	sign, callable := goja.AssertFunction(b.runtime.Get("jeth.unlockAccount"))
	if !callable {
		throwJSException(b.runtime, "jeth.unlockAccount is not callable")
	}
	val, err := sign(goja.Null(), message, account, passwd)
	if err != nil {
		throwJSException(b.runtime, err.Error())
	}
	return val
}

// Sleep will block the console for the specified number of seconds.
func (b *bridge) Sleep(call goja.FunctionCall) (response goja.Value) {
	if IsNumber(call.Argument(0)) {
		sleep := call.Argument(0).ToInteger()
		time.Sleep(time.Duration(sleep) * time.Second)
		return b.runtime.ToValue(true)
	}
	return throwJSException(b.runtime, "usage: sleep(<number of seconds>)")
}

// SleepBlocks will block the console for a specified number of new blocks optionally
// until the given timeout is reached.
func (b *bridge) SleepBlocks(call goja.FunctionCall) (response goja.Value) {
	var (
		blocks = int64(0)
		sleep  = int64(9999999999999999) // indefinitely
	)
	// Parse the input parameters for the sleep
	nArgs := len(call.Arguments)
	if nArgs == 0 {
		throwJSException(b.runtime, "usage: sleepBlocks(<n blocks>[, max sleep in seconds])")
	}
	if nArgs >= 1 {
		if IsNumber(call.Argument(0)) {
			blocks = call.Argument(0).ToInteger()
		} else {
			throwJSException(b.runtime, "expected number as first argument")
		}
	}
	if nArgs >= 2 {
		if IsNumber(call.Argument(1)) {
			sleep = call.Argument(1).ToInteger()
		} else {
			throwJSException(b.runtime, "expected number as second argument")
		}
	}
	// go through the console, this will allow web3 to call the appropriate
	// callbacks if a delayed response or notification is received.
	blockNumber := func() int64 {
		blockNumber, isFunc := goja.AssertFunction(b.runtime.Get("eth.blockNumber"))
		if !isFunc {
			throwJSException(b.runtime, "eth.blockNumber isn't a function")
		}
		block, err := blockNumber(goja.Null())
		if err != nil {
			throwJSException(b.runtime, err.Error())
		}
		// XXX This will return 0 if blockNumber isn't an Integer. This is
		// actually consistent with the current behavior (block number is 0
		// until the sync is done) but not safe enough.
		return block.ToInteger()
	}
	// Poll the current block number until either it ot a timeout is reached
	targetBlockNr := blockNumber() + blocks
	deadline := time.Now().Add(time.Duration(sleep) * time.Second)

	for time.Now().Before(deadline) {
		if blockNumber() >= targetBlockNr {
			return b.runtime.ToValue(true)
		}
		time.Sleep(time.Second)
	}
	return b.runtime.ToValue(false)
}

type jsonrpcCall struct {
	ID     int64
	Method string
	Params []interface{}
}

// Send implements the web3 provider "send" method.
func (b *bridge) Send(call goja.FunctionCall) (response goja.Value) {
	// Remarshal the request into a Go value.
	stringify, isFunc := goja.AssertFunction(b.runtime.Get("JSON.stringify"))
	if !isFunc {
		throwJSException(b.runtime, "JSON.stringify isn't a function")
	}
	reqVal, err := stringify(call.Argument(0))
	if err != nil {
		throwJSException(b.runtime, err.Error())
	}
	var (
		rawReq = reqVal.String()
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
		v, _ := b.runtime.RunString(`({"jsonrpc":"2.0"})`)
		resp := v.ToObject(b.runtime)
		resp.Set("id", req.ID)
		var result json.RawMessage
		err = b.client.Call(&result, req.Method, req.Params...)
		switch err := err.(type) {
		case nil:
			if result == nil {
				// Special case null because it is decoded as an empty
				// raw message for some reason.
				resp.Set("result", goja.Null())
			} else {
				parse, isFunc := goja.AssertFunction(b.runtime.Get("JSON.parse"))
				if !isFunc {
					throwJSException(b.runtime, "JSON.parse isn't a function")
				}
				resultVal, err := parse(b.runtime.ToValue(string(result)))
				if err != nil {
					setError(resp, -32603, err.Error())
				} else {
					resp.Set("result", resultVal)
				}
			}
		case rpc.Error:
			setError(resp, err.ErrorCode(), err.Error())
		default:
			setError(resp, -32603, err.Error())
		}
		resps = append(resps, resp)
	}

	// Return the responses either to the callback (if supplied)
	// or directly as the return value.
	if batch {
		response = b.runtime.ToValue(resps)
	} else {
		response = resps[0]
	}
	if fn, isFunc := goja.AssertFunction(call.Argument(1)); isFunc {
		fn(goja.Null(), goja.Null(), response)
		return goja.Undefined()
	}
	return response
}

func setError(resp *goja.Object, code int, msg string) {
	resp.Set("error", map[string]interface{}{"code": code, "message": msg})
}

// throwJSException panics on an goja.Value. The goja VM will recover from the
// Go panic and throw msg as a JavaScript error.
func throwJSException(runtime *goja.Runtime, msg interface{}) goja.Value {
	val := runtime.ToValue(msg)
	panic(val)
}
