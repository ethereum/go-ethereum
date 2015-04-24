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
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/natspec"
	"github.com/ethereum/go-ethereum/eth"
	re "github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
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
	re         *re.JSRE
	ethereum   *eth.Ethereum
	xeth       *xeth.XEth
	ps1        string
	atexit     func()
	corsDomain string
	prompter
}

func newJSRE(ethereum *eth.Ethereum, libPath string, interactive bool, corsDomain string) *jsre {
	js := &jsre{ethereum: ethereum, ps1: "> "}
	// set default cors domain used by startRpc from CLI flag
	js.corsDomain = corsDomain
	js.xeth = xeth.New(ethereum, js)
	js.re = re.New(libPath)
	js.apiBindings()
	js.adminBindings()

	if !liner.TerminalSupported() || !interactive {
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

func (js *jsre) apiBindings() {

	ethApi := rpc.NewEthereumApi(js.xeth)
	//js.re.Bind("jeth", rpc.NewJeth(ethApi, js.re.ToVal))

	jeth := rpc.NewJeth(ethApi, js.re.ToVal, js.re)
	//js.re.Bind("jeth", jeth)
	js.re.Set("jeth", struct{}{})
	t, _ := js.re.Get("jeth")
	jethObj := t.Object()
	jethObj.Set("send", jeth.Send)

	err := js.re.Compile("bignumber.js", re.BigNumber_JS)
	if err != nil {
		utils.Fatalf("Error loading bignumber.js: %v", err)
	}

	// we need to declare a dummy setTimeout. Otto does not support it
	_, err = js.re.Eval("setTimeout = function(cb, delay) {};")
	if err != nil {
		utils.Fatalf("Error defining setTimeout: %v", err)
	}

	err = js.re.Compile("ethereum.js", re.Ethereum_JS)
	if err != nil {
		utils.Fatalf("Error loading ethereum.js: %v", err)
	}

	_, err = js.re.Eval("var web3 = require('web3');")
	if err != nil {
		utils.Fatalf("Error requiring web3: %v", err)
	}

	_, err = js.re.Eval("web3.setProvider(jeth)")
	if err != nil {
		utils.Fatalf("Error setting web3 provider: %v", err)
	}
	_, err = js.re.Eval(`
var eth = web3.eth;
var shh = web3.shh;
var db  = web3.db;
var net = web3.net;
  `)
	if err != nil {
		utils.Fatalf("Error setting namespaces: %v", err)
	}

	js.re.Eval(globalRegistrar + "registrar = new GlobalRegistrar(\"" + globalRegistrarAddr + "\");")
}

var ds, _ = docserver.New(utils.JSpathFlag.String())

func (self *jsre) ConfirmTransaction(tx string) bool {
	if self.ethereum.NatSpec {
		notice := natspec.GetNotice(self.xeth, tx, ds)
		fmt.Println(notice)
		answer, _ := self.Prompt("Confirm Transaction\n[y/n] ")
		return strings.HasPrefix(strings.Trim(answer, " "), "y")
	} else {
		return true
	}
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
	if err := self.re.Exec(filename); err != nil {
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
		if ottoErr, ok := err.(*otto.Error); ok {
			fmt.Println(ottoErr.String())
		} else {
			fmt.Println(err)
		}
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
	val, err := self.re.PrettyPrint(v)
	if err == nil {
		fmt.Printf("%v", val)
	}
}
