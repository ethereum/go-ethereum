// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
	"github.com/peterh/liner"
)

type prompter interface {
	AppendHistory(string)
	Prompt(p string) (string, error)
	PasswordPrompt(p string) (string, error)
}

type dumbterm struct{ r *bufio.Reader }

func (r dumbterm) Prompt(p string) (string, error) {
	fmt.Print(p)
	return r.r.ReadString('\n')
}

func (r dumbterm) PasswordPrompt(p string) (string, error) {
	fmt.Println("!! Unsupported terminal, password will echo.")
	fmt.Print(p)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	return input, err
}

func (r dumbterm) AppendHistory(string) {}

type jsre struct {
	re       *javascript.JSRE
	ethereum *eth.Ethereum
	xeth     *xeth.XEth
	ps1      string
	atexit   func()

	prompter
}

func newJSRE(ethereum *eth.Ethereum) *jsre {
	js := &jsre{ethereum: ethereum, ps1: "> "}
	js.xeth = xeth.New(ethereum, js)
	js.re = javascript.NewJSRE(js.xeth)
	js.initStdFuncs()

	if !liner.TerminalSupported() {
		js.prompter = dumbterm{bufio.NewReader(os.Stdin)}
	} else {
		lr := liner.NewLiner()
		js.withHistory(func(hist *os.File) { lr.ReadHistory(hist) })
		lr.SetCtrlCAborts(true)
		js.prompter = lr
		js.atexit = func() {
			js.withHistory(func(hist *os.File) { hist.Truncate(0); lr.WriteHistory(hist) })
			lr.Close()
		}
	}
	return js
}

func (self *jsre) ConfirmTransaction(tx *types.Transaction) bool {
	p := fmt.Sprintf("Confirm Transaction %v\n[y/n] ", tx)
	answer, _ := self.Prompt(p)
	return strings.HasPrefix(strings.Trim(answer, " "), "y")
}

func (self *jsre) UnlockAccount(addr []byte) bool {
	fmt.Printf("Please unlock account %x.\n", addr)
	pass, err := self.PasswordPrompt("Passphrase: ")
	if err != nil {
		return false
	}
	// TODO: allow retry
	if err := self.ethereum.AccountManager().Unlock(addr, pass); err != nil {
		return false
	} else {
		fmt.Println("Account is now unlocked for this session.")
		return true
	}
}

func (self *jsre) exec(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	if _, err := self.re.Run(string(content)); err != nil {
		return fmt.Errorf("Javascript Error: %v", err)
	}
	return nil
}

func (self *jsre) interactive() {
	for {
		input, err := self.Prompt(self.ps1)
		if err != nil {
			break
		}
		if input == "" {
			continue
		}
		str += input + "\n"
		self.setIndent()
		if indentCount <= 0 {
			if input == "exit" {
				break
			}
			hist := str[:len(str)-1]
			self.AppendHistory(hist)
			self.parseInput(str)
			str = ""
		}
	}
	if self.atexit != nil {
		self.atexit()
	}
}

func (self *jsre) withHistory(op func(*os.File)) {
	hist, err := os.OpenFile(path.Join(self.ethereum.DataDir, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Printf("unable to open history file: %v\n", err)
		return
	}
	op(hist)
	hist.Close()
}

func (self *jsre) parseInput(code string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[native] error", r)
		}
	}()
	value, err := self.re.Run(code)
	if err != nil {
		fmt.Println(err)
		return
	}
	self.printValue(value)
}

var indentCount = 0
var str = ""

func (self *jsre) setIndent() {
	open := strings.Count(str, "{")
	open += strings.Count(str, "(")
	closed := strings.Count(str, "}")
	closed += strings.Count(str, ")")
	indentCount = open - closed
	if indentCount <= 0 {
		self.ps1 = "> "
	} else {
		self.ps1 = strings.Join(make([]string, indentCount*2), "..")
		self.ps1 += " "
	}
}

func (self *jsre) printValue(v interface{}) {
	method, _ := self.re.Vm.Get("prettyPrint")
	v, err := self.re.Vm.ToValue(v)
	if err == nil {
		val, err := method.Call(method, v)
		if err == nil {
			fmt.Printf("%v", val)
		}
	}
}

func (self *jsre) initStdFuncs() {
	t, _ := self.re.Vm.Get("eth")
	eth := t.Object()
	eth.Set("connect", self.connect)
	eth.Set("stopMining", self.stopMining)
	eth.Set("startMining", self.startMining)
	eth.Set("dump", self.dump)
	eth.Set("export", self.export)
}

/*
 * The following methods are natively implemented javascript functions.
 */

func (self *jsre) dump(call otto.FunctionCall) otto.Value {
	var block *types.Block

	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = self.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = self.ethereum.ChainManager().GetBlock(common.Hex2Bytes(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

		if block == nil {
			fmt.Println("block not found")

			return otto.UndefinedValue()
		}

	} else {
		block = self.ethereum.ChainManager().CurrentBlock()
	}

	statedb := state.New(block.Root(), self.ethereum.StateDb())

	v, _ := self.re.Vm.ToValue(statedb.RawDump())

	return v
}

func (self *jsre) stopMining(call otto.FunctionCall) otto.Value {
	self.ethereum.StopMining()
	return otto.TrueValue()
}

func (self *jsre) startMining(call otto.FunctionCall) otto.Value {
	if err := self.ethereum.StartMining(); err != nil {
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *jsre) connect(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	if err := self.ethereum.SuggestPeer(nodeURL); err != nil {
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *jsre) export(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	data := self.ethereum.ChainManager().Export()

	if err := common.WriteFile(fn, data); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}
