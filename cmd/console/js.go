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
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/docserver"
	re "github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc"
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
	line, err := r.r.ReadString('\n')
	return strings.TrimSuffix(line, "\n"), err
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
	re      *re.JSRE
	wait    chan *big.Int
	ps1     string
	atexit  func()
	datadir string
	prompter
}

func newJSRE(libPath, ipcpath string) *jsre {
	js := &jsre{ps1: "> "}
	js.wait = make(chan *big.Int)

	// update state in separare forever blocks
	js.re = re.New(libPath)
	js.apiBindings(ipcpath)

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
			close(js.wait)
		}
	}
	return js
}

func (js *jsre) apiBindings(ipcpath string) {
	ethApi := rpc.NewEthereumApi(nil)
	jeth := rpc.NewJeth(ethApi, js.re, ipcpath)

	js.re.Set("jeth", struct{}{})
	t, _ := js.re.Get("jeth")
	jethObj := t.Object()
	jethObj.Set("send", jeth.SendIpc)
	jethObj.Set("sendAsync", jeth.SendIpc)

	err := js.re.Compile("bignumber.js", re.BigNumber_JS)
	if err != nil {
		utils.Fatalf("Error loading bignumber.js: %v", err)
	}

	err = js.re.Compile("ethereum.js", re.Web3_JS)
	if err != nil {
		utils.Fatalf("Error loading web3.js: %v", err)
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
  `)

	if err != nil {
		utils.Fatalf("Error setting namespaces: %v", err)
	}

	js.re.Eval(globalRegistrar + "registrar = GlobalRegistrar.at(\"" + globalRegistrarAddr + "\");")
}

var ds, _ = docserver.New("/")

/*
func (self *jsre) ConfirmTransaction(tx string) bool {
	if self.ethereum.NatSpec {
		notice := natspec.GetNotice(self.xeth, tx, ds)
		fmt.Println(notice)
		answer, _ := self.Prompt("Confirm Transaction [y/n]")
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
	if err := self.ethereum.AccountManager().Unlock(common.BytesToAddress(addr), pass); err != nil {
		return false
	} else {
		fmt.Println("Account is now unlocked for this session.")
		return true
	}
}
*/

func (self *jsre) exec(filename string) error {
	if err := self.re.Exec(filename); err != nil {
		self.re.Stop(false)
		return fmt.Errorf("Javascript Error: %v", err)
	}
	self.re.Stop(true)
	return nil
}

// show summary of current geth instance
func (self *jsre) welcome() {
	self.re.Eval(`
		console.log('Connected to ' + web3.version.client);
	`)
}

func (self *jsre) interactive() {
	// Read input lines.
	prompt := make(chan string)
	inputln := make(chan string)
	go func() {
		defer close(inputln)
		for {
			line, err := self.Prompt(<-prompt)
			if err != nil {
				return
			}
			inputln <- line
		}
	}()
	// Wait for Ctrl-C, too.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	defer func() {
		if self.atexit != nil {
			self.atexit()
		}
		self.re.Stop(false)
	}()
	for {
		prompt <- self.ps1
		select {
		case <-sig:
			fmt.Println("caught interrupt, exiting")
			return
		case input, ok := <-inputln:
			if !ok || indentCount <= 0 && input == "exit" {
				return
			}
			if input == "" {
				continue
			}
			str += input + "\n"
			self.setIndent()
			if indentCount <= 0 {
				hist := str[:len(str)-1]
				self.AppendHistory(hist)
				self.parseInput(str)
				str = ""
			}
		}
	}
}

func (self *jsre) withHistory(op func(*os.File)) {
	hist, err := os.OpenFile(filepath.Join(self.datadir, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
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
