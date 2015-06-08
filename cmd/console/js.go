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

	"encoding/json"

	"sort"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/docserver"
	re "github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
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

var (
	loadedModulesMethods map[string][]string
)

func loadAutoCompletion(js *jsre, ipcpath string) {
	modules, err := js.suportedApis(ipcpath)
	if err != nil {
		utils.Fatalf("Unable to determine supported modules - %v", err)
	}

	fmt.Printf("load autocompletion %v", modules)

	loadedModulesMethods = make(map[string][]string)
	for module, _ := range modules {
		loadedModulesMethods[module] = api.AutoCompletion[module]
	}
}

func keywordCompleter(line string) []string {
	results := make([]string, 0)

	if strings.Contains(line, ".") {
		elements := strings.Split(line, ".")
		if len(elements) == 2 {
			module := elements[0]
			partialMethod := elements[1]
			if methods, found := loadedModulesMethods[module]; found {
				for _, method := range methods {
					if strings.HasPrefix(method, partialMethod) { // e.g. debug.se
						results = append(results, module+"."+method)
					}
				}
			}
		}
	} else {
		for module, methods := range loadedModulesMethods {
			if line == module { // user typed in full module name, show all methods
				for _, method := range methods {
					results = append(results, module+"."+method)
				}
			} else if strings.HasPrefix(module, line) { // partial method name, e.g. admi
				results = append(results, module)
			}
		}
	}
	return results
}

func apiWordCompleter(line string, pos int) (head string, completions []string, tail string) {
	if len(line) == 0 {
		return "", nil, ""
	}

	i := 0
	for i = pos - 1; i > 0; i-- {
		if line[i] == '.' || (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') {
			continue
		}
		if i >= 3 && line[i] == '3' && line[i-3] == 'w' && line[i-2] == 'e' && line[i-1] == 'b' {
			continue
		}
		i += 1
		break
	}

	begin := line[:i]
	keyword := line[i:pos]
	end := line[pos:]

	completionWords := keywordCompleter(keyword)
	return begin, completionWords, end
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
		loadAutoCompletion(js, ipcpath)
		lr.SetWordCompleter(apiWordCompleter)
		lr.SetTabCompletionStyle(liner.TabPrints)
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

	apis, err := js.suportedApis(ipcpath)
	if err != nil {
		utils.Fatalf("Unable to determine supported api's: %v", err)
	}

	// load only supported API's in javascript runtime
	shortcuts := "var eth = web3.eth; "
	for apiName, _ := range apis {
		if apiName == api.Web3ApiName || apiName == api.EthApiName {
			continue // manually mapped
		}

		if err = js.re.Compile(fmt.Sprintf("%s.js", apiName), api.Javascript(apiName)); err == nil {
			shortcuts += fmt.Sprintf("var %s = web3.%s; ", apiName, apiName)
		} else {
			utils.Fatalf("Error loading %s.js: %v", apiName, err)
		}
	}

	_, err = js.re.Eval(shortcuts)

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

func (self *jsre) suportedApis(ipcpath string) (map[string]string, error) {
	config := comms.IpcConfig{
		Endpoint: ipcpath,
	}

	client, err := comms.NewIpcClient(config, codec.JSON)
	if err != nil {
		return nil, err
	}

	req := shared.Request{
		Id:      1,
		Jsonrpc: "2.0",
		Method:  "modules",
	}

	err = client.Send(req)
	if err != nil {
		return nil, err
	}

	res, err := client.Recv()
	if err != nil {
		return nil, err
	}

	if sucRes, ok := res.(shared.SuccessResponse); ok {
		data, _ := json.Marshal(sucRes.Result)
		apis := make(map[string]string)
		err = json.Unmarshal(data, &apis)
		if err == nil {
			return apis, nil
		}
	}

	return nil, fmt.Errorf("Unable to determine supported API's")
}

// show summary of current geth instance
func (self *jsre) welcome(ipcpath string) {
	self.re.Eval(`console.log('instance: ' + web3.version.client);`)
	self.re.Eval(`console.log(' datadir: ' + admin.datadir);`)
	self.re.Eval(`console.log("coinbase: " + eth.coinbase);`)
	self.re.Eval(`var lastBlockTimestamp = 1000 * eth.getBlock(eth.blockNumber).timestamp`)
	self.re.Eval(`console.log("at block: " + eth.blockNumber + " (" + new Date(lastBlockTimestamp).toLocaleDateString()
		+ " " + new Date(lastBlockTimestamp).toLocaleTimeString() + ")");`)

	if modules, err := self.suportedApis(ipcpath); err == nil {
		loadedModules := make([]string, 0)
		for api, version := range modules {
			loadedModules = append(loadedModules, fmt.Sprintf("%s:%s", api, version))
		}
		sort.Strings(loadedModules)

		self.re.Eval(fmt.Sprintf("var modules = '%s';", strings.Join(loadedModules, " ")))
		self.re.Eval(`console.log(" modules: " + modules);`)
	}
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
