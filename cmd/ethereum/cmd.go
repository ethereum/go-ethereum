/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import (
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/cmd/ethereum/repl"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/javascript"
)

func InitJsConsole(ethereum *eth.Ethereum) {
	repl := ethrepl.NewJSRepl(ethereum)
	go repl.Start()
	utils.RegisterInterrupt(func(os.Signal) {
		repl.Stop()
	})
}

func ExecJsFile(ethereum *eth.Ethereum, InputFile string) {
	file, err := os.Open(InputFile)
	if err != nil {
		clilogger.Fatalln(err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		clilogger.Fatalln(err)
	}
	re := javascript.NewJSRE(ethereum)
	utils.RegisterInterrupt(func(os.Signal) {
		re.Stop()
	})
	re.Run(string(content))
}
