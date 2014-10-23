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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethlog"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/javascript"
)

var logger = ethlog.NewLogger("REPL")

type Repl interface {
	Start()
	Stop()
}

type JSRepl struct {
	re *javascript.JSRE

	prompt string

	history *os.File

	running bool
}

func NewJSRepl(ethereum *eth.Ethereum) *JSRepl {
	hist, err := os.OpenFile(path.Join(ethutil.Config.ExecPath, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}

	return &JSRepl{re: javascript.NewJSRE(ethereum), prompt: "> ", history: hist}
}

func (self *JSRepl) Start() {
	if !self.running {
		self.running = true
		logger.Infoln("init JS Console")
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
		self.re.Stop()
		logger.Infoln("exit JS Console")
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
