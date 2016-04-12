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
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/eth"
	re "github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/peterh/liner"
	"github.com/robertkrimen/otto"
)

var (
	passwordRegexp = regexp.MustCompile("personal.[nu]")
	leadingSpace   = regexp.MustCompile("^ ")
	onlyws         = regexp.MustCompile("^\\s*$")
	exit           = regexp.MustCompile("^\\s*exit\\s*;*\\s*$")
)

type jsre struct {
	re         *re.JSRE
	stack      *node.Node
	wait       chan *big.Int
	ps1        string
	atexit     func()
	corsDomain string
	client     rpc.Client
}

func makeCompleter(re *jsre) liner.WordCompleter {
	return func(line string, pos int) (head string, completions []string, tail string) {
		if len(line) == 0 || pos == 0 {
			return "", nil, ""
		}
		// chuck data to relevant part for autocompletion, e.g. in case of nested lines eth.getBalance(eth.coinb<tab><tab>
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
		return line[:i], re.re.CompleteKeywords(line[i:pos]), line[pos:]
	}
}

func newLightweightJSRE(docRoot string, client rpc.Client, datadir string, interactive bool) *jsre {
	js := &jsre{ps1: "> "}
	js.wait = make(chan *big.Int)
	js.client = client
	js.re = re.New(docRoot)
	if err := js.apiBindings(); err != nil {
		utils.Fatalf("Unable to initialize console - %v", err)
	}
	js.setupInput(datadir)
	return js
}

func newJSRE(stack *node.Node, docRoot, corsDomain string, client rpc.Client, interactive bool) *jsre {
	js := &jsre{stack: stack, ps1: "> "}
	// set default cors domain used by startRpc from CLI flag
	js.corsDomain = corsDomain
	js.wait = make(chan *big.Int)
	js.client = client
	js.re = re.New(docRoot)
	if err := js.apiBindings(); err != nil {
		utils.Fatalf("Unable to connect - %v", err)
	}
	js.setupInput(stack.DataDir())
	return js
}

func (self *jsre) setupInput(datadir string) {
	self.withHistory(datadir, func(hist *os.File) { utils.Stdin.ReadHistory(hist) })
	utils.Stdin.SetCtrlCAborts(true)
	utils.Stdin.SetWordCompleter(makeCompleter(self))
	utils.Stdin.SetTabCompletionStyle(liner.TabPrints)
	self.atexit = func() {
		self.withHistory(datadir, func(hist *os.File) {
			hist.Truncate(0)
			utils.Stdin.WriteHistory(hist)
		})
		utils.Stdin.Close()
		close(self.wait)
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
      console.log('instance: ' + web3.version.node);
      console.log("coinbase: " + eth.coinbase);
      var ts = 1000 * eth.getBlock(eth.blockNumber).timestamp;
      console.log("at block: " + eth.blockNumber + " (" + new Date(ts) + ")");
      console.log(' datadir: ' + admin.datadir);
    })();
  `)
	if modules, err := self.supportedApis(); err == nil {
		loadedModules := make([]string, 0)
		for api, version := range modules {
			loadedModules = append(loadedModules, fmt.Sprintf("%s:%s", api, version))
		}
		sort.Strings(loadedModules)
	}
}

func (self *jsre) supportedApis() (map[string]string, error) {
	return self.client.SupportedModules()
}

func (js *jsre) apiBindings() error {
	apis, err := js.supportedApis()
	if err != nil {
		return err
	}

	apiNames := make([]string, 0, len(apis))
	for a, _ := range apis {
		apiNames = append(apiNames, a)
	}

	jeth := utils.NewJeth(js.re, js.client)
	js.re.Set("jeth", struct{}{})
	t, _ := js.re.Get("jeth")
	jethObj := t.Object()

	jethObj.Set("send", jeth.Send)
	jethObj.Set("sendAsync", jeth.Send)

	err = js.re.Compile("bignumber.js", re.BigNumber_JS)
	if err != nil {
		utils.Fatalf("Error loading bignumber.js: %v", err)
	}

	err = js.re.Compile("web3.js", re.Web3_JS)
	if err != nil {
		utils.Fatalf("Error loading web3.js: %v", err)
	}

	_, err = js.re.Run("var Web3 = require('web3');")
	if err != nil {
		utils.Fatalf("Error requiring web3: %v", err)
	}

	_, err = js.re.Run("var web3 = new Web3(jeth);")
	if err != nil {
		utils.Fatalf("Error setting web3 provider: %v", err)
	}

	// load only supported API's in javascript runtime
	shortcuts := "var eth = web3.eth; var personal = web3.personal; "
	for _, apiName := range apiNames {
		if apiName == "web3" || apiName == "rpc" {
			continue // manually mapped or ignore
		}

		if jsFile, ok := rpc.WEB3Extensions[apiName]; ok {
			if err = js.re.Compile(fmt.Sprintf("%s.js", apiName), jsFile); err == nil {
				shortcuts += fmt.Sprintf("var %s = web3.%s; ", apiName, apiName)
			} else {
				utils.Fatalf("Error loading %s.js: %v", apiName, err)
			}
		}
	}

	_, err = js.re.Run(shortcuts)
	if err != nil {
		utils.Fatalf("Error setting namespaces: %v", err)
	}

	js.re.Run(`var GlobalRegistrar = eth.contract(` + registrar.GlobalRegistrarAbi + `);   registrar = GlobalRegistrar.at("` + registrar.GlobalRegistrarAddr + `");`)

	// overrule some of the methods that require password as input and ask for it interactively
	p, err := js.re.Get("personal")
	if err != nil {
		fmt.Println("Unable to overrule sensitive methods in personal module")
		return nil
	}

	// Override the unlockAccount and newAccount methods on the personal object since these require user interaction.
	// Assign the jeth.unlockAccount and jeth.newAccount in the jsre the original web3 callbacks. These will be called
	// by the jeth.* methods after they got the password from the user and send the original web3 request to the backend.
	if persObj := p.Object(); persObj != nil { // make sure the personal api is enabled over the interface
		js.re.Run(`jeth.unlockAccount = personal.unlockAccount;`)
		persObj.Set("unlockAccount", jeth.UnlockAccount)
		js.re.Run(`jeth.newAccount = personal.newAccount;`)
		persObj.Set("newAccount", jeth.NewAccount)
	}

	// The admin.sleep and admin.sleepBlocks are offered by the console and not by the RPC layer.
	// Bind these if the admin module is available.
	if a, err := js.re.Get("admin"); err == nil {
		if adminObj := a.Object(); adminObj != nil {
			adminObj.Set("sleepBlocks", jeth.SleepBlocks)
			adminObj.Set("sleep", jeth.Sleep)
		}
	}

	return nil
}

func (self *jsre) AskPassword() (string, bool) {
	pass, err := utils.Stdin.PasswordPrompt("Passphrase: ")
	if err != nil {
		return "", false
	}
	return pass, true
}

func (self *jsre) ConfirmTransaction(tx string) bool {
	// Retrieve the Ethereum instance from the node
	var ethereum *eth.Ethereum
	if err := self.stack.Service(&ethereum); err != nil {
		return false
	}
	// If natspec is enabled, ask for permission
	if ethereum.NatSpec && false /* disabled for now */ {
		//		notice := natspec.GetNotice(self.xeth, tx, ethereum.HTTPClient())
		//		fmt.Println(notice)
		//		answer, _ := self.Prompt("Confirm Transaction [y/n]")
		//		return strings.HasPrefix(strings.Trim(answer, " "), "y")
	}
	return true
}

func (self *jsre) UnlockAccount(addr []byte) bool {
	fmt.Printf("Please unlock account %x.\n", addr)
	pass, err := utils.Stdin.PasswordPrompt("Passphrase: ")
	if err != nil {
		return false
	}
	// TODO: allow retry
	var ethereum *eth.Ethereum
	if err := self.stack.Service(&ethereum); err != nil {
		return false
	}
	a := accounts.Account{Address: common.BytesToAddress(addr)}
	if err := ethereum.AccountManager().Unlock(a, pass); err != nil {
		return false
	} else {
		fmt.Println("Account is now unlocked for this session.")
		return true
	}
}

// preloadJSFiles loads JS files that the user has specified with ctx.PreLoadJSFlag into
// the JSRE. If not all files could be loaded it will return an error describing the error.
func (self *jsre) preloadJSFiles(ctx *cli.Context) error {
	if ctx.GlobalString(utils.PreLoadJSFlag.Name) != "" {
		assetPath := ctx.GlobalString(utils.JSpathFlag.Name)
		jsFiles := strings.Split(ctx.GlobalString(utils.PreLoadJSFlag.Name), ",")
		for _, file := range jsFiles {
			filename := common.AbsolutePath(assetPath, strings.TrimSpace(file))
			if err := self.re.Exec(filename); err != nil {
				return fmt.Errorf("%s: %v", file, err)
			}
		}
	}
	return nil
}

// exec executes the JS file with the given filename and stops the JSRE
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
			line, err := utils.Stdin.Prompt(<-prompt)
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
					utils.Stdin.AppendHistory(str[:len(str)-1])
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
