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

package ethrepl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

var repllogger = logger.NewLogger("REPL")

type Repl interface {
	Start()
	Stop()
}

type JSRepl struct {
	re       *javascript.JSRE
	ethereum *eth.Ethereum
	xeth     *xeth.XEth

	prompt string

	history *os.File

	running bool
}

func NewJSRepl(ethereum *eth.Ethereum) *JSRepl {
	hist, err := os.OpenFile(path.Join(ethutil.Config.ExecPath, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}

	xeth := xeth.New(ethereum)
	repl := &JSRepl{re: javascript.NewJSRE(xeth), xeth: xeth, ethereum: ethereum, prompt: "> ", history: hist}
	repl.initStdFuncs()

	return repl
}

func (self *JSRepl) Start() {
	if !self.running {
		self.running = true
		repllogger.Infoln("init JS Console")

		reader := bufio.NewReader(self.history)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("error reading history", err)
				break
			}

			addHistory(line[:len(line)-1])
		}
		self.read()
	}
}

func (self *JSRepl) Stop() {
	if self.running {
		self.running = false
		repllogger.Infoln("exit JS Console")
		self.history.Close()
	}
}

func (self *JSRepl) parseInput(code string) {
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

	self.PrintValue(value)
}

func (self *JSRepl) initStdFuncs() {
	t, _ := self.re.Vm.Get("eth")
	eth := t.Object()
	eth.Set("connect", self.connect)
	eth.Set("stopMining", self.stopMining)
	eth.Set("startMining", self.startMining)
	eth.Set("dump", self.dump)
	eth.Set("export", self.export)
}

/*
 * The following methods are natively implemented javascript functions
 */

func (self *JSRepl) dump(call otto.FunctionCall) otto.Value {
	var block *types.Block

	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = self.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = self.ethereum.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
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

	statedb := state.New(block.Root(), self.ethereum.Db())

	v, _ := self.re.Vm.ToValue(statedb.RawDump())

	return v
}

func (self *JSRepl) stopMining(call otto.FunctionCall) otto.Value {
	self.xeth.Miner().Stop()

	return otto.TrueValue()
}

func (self *JSRepl) startMining(call otto.FunctionCall) otto.Value {
	self.xeth.Miner().Start()
	return otto.TrueValue()
}

func (self *JSRepl) connect(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	if err := self.ethereum.SuggestPeer(nodeURL); err != nil {
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *JSRepl) export(call otto.FunctionCall) otto.Value {
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

	if err := ethutil.WriteFile(fn, data); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}
