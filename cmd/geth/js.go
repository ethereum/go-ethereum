// Copyright 2014 The go-ethereum Authors
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

package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"sort"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/natspec"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/eth"
	re "github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)

var (
	passwordRegexp = regexp.MustCompile("personal.[nu]")
	leadingSpace   = regexp.MustCompile("^ ")
	onlyws         = regexp.MustCompile("^\\s*$")
	exit           = regexp.MustCompile("^\\s*exit\\s*;*\\s*$")
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
	re         *re.JSRE
	stack      *node.Node
	xeth       *xeth.XEth
	wait       chan *big.Int
	ps1        string
	atexit     func()
	corsDomain string
	client     comms.EthereumClient
	prompter
}

var (
	loadedModulesMethods map[string][]string
)

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
	if len(line) == 0 || pos == 0 {
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

func newLightweightJSRE(docRoot string, client comms.EthereumClient, datadir string, interactive bool) *jsre {
	js := &jsre{ps1: "> "}
	js.wait = make(chan *big.Int)
	js.client = client

	// update state in separare forever blocks
	js.re = re.New(docRoot)
	if err := js.apiBindings(js); err != nil {
		utils.Fatalf("Unable to initialize console - %v", err)
	}

	if !liner.TerminalSupported() || !interactive {
		js.prompter = dumbterm{bufio.NewReader(os.Stdin)}
	} else {
		lr := liner.NewLiner()
		js.withHistory(datadir, func(hist *os.File) { lr.ReadHistory(hist) })
		lr.SetCtrlCAborts(true)
		js.loadAutoCompletion()
		lr.SetWordCompleter(apiWordCompleter)
		lr.SetTabCompletionStyle(liner.TabPrints)
		js.prompter = lr
		js.atexit = func() {
			js.withHistory(datadir, func(hist *os.File) { hist.Truncate(0); lr.WriteHistory(hist) })
			lr.Close()
			close(js.wait)
		}
	}
	return js
}

func newJSRE(stack *node.Node, docRoot, corsDomain string, client comms.EthereumClient, interactive bool, f xeth.Frontend) *jsre {
	js := &jsre{stack: stack, ps1: "> "}
	// set default cors domain used by startRpc from CLI flag
	js.corsDomain = corsDomain
	if f == nil {
		f = js
	}
	js.xeth = xeth.New(stack, f)
	js.wait = js.xeth.UpdateState()
	js.client = client
	if clt, ok := js.client.(*comms.InProcClient); ok {
		if offeredApis, err := api.ParseApiString(shared.AllApis, codec.JSON, js.xeth, stack); err == nil {
			clt.Initialize(api.Merge(offeredApis...))
		}
	}

	// update state in separare forever blocks
	js.re = re.New(docRoot)
	if err := js.apiBindings(f); err != nil {
		utils.Fatalf("Unable to connect - %v", err)
	}

	if !liner.TerminalSupported() || !interactive {
		js.prompter = dumbterm{bufio.NewReader(os.Stdin)}
	} else {
		lr := liner.NewLiner()
		js.withHistory(stack.DataDir(), func(hist *os.File) { lr.ReadHistory(hist) })
		lr.SetCtrlCAborts(true)
		js.loadAutoCompletion()
		lr.SetWordCompleter(apiWordCompleter)
		lr.SetTabCompletionStyle(liner.TabPrints)
		js.prompter = lr
		js.atexit = func() {
			js.withHistory(stack.DataDir(), func(hist *os.File) { hist.Truncate(0); lr.WriteHistory(hist) })
			lr.Close()
			close(js.wait)
		}
	}
	return js
}

func (self *jsre) loadAutoCompletion() {
	if modules, err := self.supportedApis(); err == nil {
		loadedModulesMethods = make(map[string][]string)
		for module, _ := range modules {
			loadedModulesMethods[module] = api.AutoCompletion[module]
		}
	}
}

func (self *jsre) batch(statement string) {
	err := self.re.EvalAndPrettyPrint(statement)

	if err != nil {
		fmt.Printf("error: %v", err)
	}

	if self.atexit != nil {
		self.atexit()
	}

	self.re.Stop(false)
}

// show summary of current geth instance
func (self *jsre) welcome() {
	self.re.Run(`
    (function () {
      console.log('instance: ' + web3.version.client);
      console.log(' datadir: ' + admin.datadir);
      console.log("coinbase: " + eth.coinbase);
      var ts = 1000 * eth.getBlock(eth.blockNumber).timestamp;
      console.log("at block: " + eth.blockNumber + " (" + new Date(ts) + ")");
    })();
  `)
	if modules, err := self.supportedApis(); err == nil {
		loadedModules := make([]string, 0)
		for api, version := range modules {
			loadedModules = append(loadedModules, fmt.Sprintf("%s:%s", api, version))
		}
		sort.Strings(loadedModules)
		fmt.Println("modules:", strings.Join(loadedModules, " "))
	}
}

func (self *jsre) supportedApis() (map[string]string, error) {
	return self.client.SupportedModules()
}

func (js *jsre) apiBindings(f xeth.Frontend) error {
	apis, err := js.supportedApis()
	if err != nil {
		return err
	}

	apiNames := make([]string, 0, len(apis))
	for a, _ := range apis {
		apiNames = append(apiNames, a)
	}

	apiImpl, err := api.ParseApiString(strings.Join(apiNames, ","), codec.JSON, js.xeth, js.stack)
	if err != nil {
		utils.Fatalf("Unable to determine supported api's: %v", err)
	}

	jeth := rpc.NewJeth(api.Merge(apiImpl...), js.re, js.client, f)
	js.re.Set("jeth", struct{}{})
	t, _ := js.re.Get("jeth")
	jethObj := t.Object()

	jethObj.Set("send", jeth.Send)
	jethObj.Set("sendAsync", jeth.Send)

	err = js.re.Compile("bignumber.js", re.BigNumber_JS)
	if err != nil {
		utils.Fatalf("Error loading bignumber.js: %v", err)
	}

	err = js.re.Compile("ethereum.js", re.Web3_JS)
	if err != nil {
		utils.Fatalf("Error loading web3.js: %v", err)
	}

	_, err = js.re.Run("var web3 = require('web3');")
	if err != nil {
		utils.Fatalf("Error requiring web3: %v", err)
	}

	_, err = js.re.Run("web3.setProvider(jeth)")
	if err != nil {
		utils.Fatalf("Error setting web3 provider: %v", err)
	}

	// load only supported API's in javascript runtime
	shortcuts := "var eth = web3.eth; "
	for _, apiName := range apiNames {
		if apiName == shared.Web3ApiName {
			continue // manually mapped
		}

		if err = js.re.Compile(fmt.Sprintf("%s.js", apiName), api.Javascript(apiName)); err == nil {
			shortcuts += fmt.Sprintf("var %s = web3.%s; ", apiName, apiName)
		} else {
			utils.Fatalf("Error loading %s.js: %v", apiName, err)
		}
	}

	_, err = js.re.Run(shortcuts)

	if err != nil {
		utils.Fatalf("Error setting namespaces: %v", err)
	}

	js.re.Run(`var GlobalRegistrar = eth.contract(` + registrar.GlobalRegistrarAbi + `);   registrar = GlobalRegistrar.at("` + registrar.GlobalRegistrarAddr + `");`)
	return nil
}

func (self *jsre) AskPassword() (string, bool) {
	pass, err := self.PasswordPrompt("Passphrase: ")
	if err != nil {
		return "", false
	}
	return pass, true
}

func (self *jsre) ConfirmTransaction(tx string) bool {
	// Retrieve the Ethereum instance from the node
	var ethereum *eth.Ethereum
	if _, err := self.stack.SingletonService(&ethereum); err != nil {
		return false
	}
	// If natspec is enabled, ask for permission
	if ethereum.NatSpec {
		notice := natspec.GetNotice(self.xeth, tx, ethereum.HTTPClient())
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
	var ethereum *eth.Ethereum
	if _, err := self.stack.SingletonService(&ethereum); err != nil {
		return false
	}
	if err := ethereum.AccountManager().Unlock(common.BytesToAddress(addr), pass); err != nil {
		return false
	} else {
		fmt.Println("Account is now unlocked for this session.")
		return true
	}
}

func (self *jsre) exec(filename string) error {
	if err := self.re.Exec(filename); err != nil {
		self.re.Stop(false)
		return fmt.Errorf("Javascript Error: %v", err)
	}
	self.re.Stop(true)
	return nil
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
				if err == liner.ErrPromptAborted { // ctrl-C
					self.resetPrompt()
					inputln <- ""
					continue
				}
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
			if !ok || indentCount <= 0 && exit.MatchString(input) {
				return
			}
			if onlyws.MatchString(input) {
				continue
			}
			str += input + "\n"
			self.setIndent()
			if indentCount <= 0 {
				if mustLogInHistory(str) {
					self.AppendHistory(str[:len(str)-1])
				}
				self.parseInput(str)
				str = ""
			}
		}
	}
}

func mustLogInHistory(input string) bool {
	return len(input) == 0 ||
		passwordRegexp.MatchString(input) ||
		!leadingSpace.MatchString(input)
}

func (self *jsre) withHistory(datadir string, op func(*os.File)) {
	hist, err := os.OpenFile(filepath.Join(datadir, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
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
	if err := self.re.EvalAndPrettyPrint(code); err != nil {
		if ottoErr, ok := err.(*otto.Error); ok {
			fmt.Println(ottoErr.String())
		} else {
			fmt.Println(err)
		}
		return
	}
}

var indentCount = 0
var str = ""

func (self *jsre) resetPrompt() {
	indentCount = 0
	str = ""
	self.ps1 = "> "
}

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
