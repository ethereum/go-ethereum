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
	"os/signal"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/jethre"
	"github.com/peterh/liner"
)

func execJsFile(ethereum *eth.Ethereum, assetPath, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	re := javascript.NewJEthRE(ethereum, assetPath)
	if _, err := re.Run(string(content)); err != nil {
		utils.Fatalf("Javascript Error: %v", err)
	}
}

type repl struct {
	re      *javascript.JEthRE
	prompt  string
	lr      *liner.State
	dataDir string
}

func runREPL(ethereum *eth.Ethereum, assetPath string) {
	repl := &repl{
		re:      javascript.NewJEthRE(ethereum, assetPath),
		dataDir: ethereum.DataDir,
		prompt:  "> ",
	}
	if !liner.TerminalSupported() {
		repl.dumbRead()
	} else {
		lr := liner.NewLiner()
		defer lr.Close()
		lr.SetCtrlCAborts(true)
		repl.withHistory(func(hist *os.File) { lr.ReadHistory(hist) })
		repl.read(lr)
		repl.withHistory(func(hist *os.File) { hist.Truncate(0); lr.WriteHistory(hist) })
	}
}

func (self *repl) withHistory(op func(*os.File)) {
	hist, err := os.OpenFile(path.Join(self.dataDir, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Printf("unable to open history file: %v\n", err)
		return
	}
	op(hist)
	hist.Close()
}

func (self *repl) parseInput(code string) {
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

func (self *repl) setIndent() {
	open := strings.Count(str, "{")
	open += strings.Count(str, "(")
	closed := strings.Count(str, "}")
	closed += strings.Count(str, ")")
	indentCount = open - closed
	if indentCount <= 0 {
		self.prompt = "> "
	} else {
		self.prompt = strings.Join(make([]string, indentCount*2), "..")
		self.prompt += " "
	}
}

func (self *repl) read(lr *liner.State) {
	for {
		input, err := lr.Prompt(self.prompt)
		if err != nil {
			return
		}
		if input == "" {
			continue
		}
		str += input + "\n"
		self.setIndent()
		if indentCount <= 0 {
			if input == "exit" {
				return
			}
			hist := str[:len(str)-1]
			lr.AppendHistory(hist)
			self.parseInput(str)
			str = ""
		}
	}
}

func (self *repl) dumbRead() {
	fmt.Println("Unsupported terminal, line editing will not work.")

	// process lines
	readDone := make(chan struct{})
	go func() {
		r := bufio.NewReader(os.Stdin)
	loop:
		for {
			fmt.Print(self.prompt)
			line, err := r.ReadString('\n')
			switch {
			case err != nil || line == "exit":
				break loop
			case line == "":
				continue
			default:
				self.parseInput(line + "\n")
			}
		}
		close(readDone)
	}()

	// wait for Ctrl-C
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)
	defer signal.Stop(sigc)

	select {
	case <-readDone:
	case <-sigc:
		os.Stdin.Close() // terminate read
	}
}

func (self *repl) printValue(v interface{}) {
	val, err := self.re.PrettyPrint(v)
	if err == nil {
		fmt.Printf("%v", val)
	} else {
		fmt.Printf("print error: %v", err)
	}
}
