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

// #cgo darwin CFLAGS: -I/usr/local/opt/readline/include
// #cgo darwin LDFLAGS: -L/usr/local/opt/readline/lib
// #cgo LDFLAGS: -lreadline
// #include <stdio.h>
// #include <stdlib.h>
// #include <readline/readline.h>
// #include <readline/history.h>
import "C"
import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unsafe"
)

func initReadLine() {
	C.rl_catch_sigwinch = 0
	C.rl_catch_signals = 0
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGWINCH:
				C.rl_resize_terminal()

			case os.Interrupt:
				C.rl_cleanup_after_signal()
			default:

			}
		}
	}()
}

func readLine(prompt *string) *string {
	var p *C.char

	//readline allows an empty prompt(NULL)
	if prompt != nil {
		p = C.CString(*prompt)
	}

	ret := C.readline(p)

	if p != nil {
		C.free(unsafe.Pointer(p))
	}

	if ret == nil {
		return nil
	} //EOF

	s := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &s
}

func addHistory(s string) {
	p := C.CString(s)
	C.add_history(p)
	C.free(unsafe.Pointer(p))
}

var indentCount = 0
var str = ""

func (self *JSRepl) setIndent() {
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

func (self *JSRepl) read() {
	initReadLine()
L:
	for {
		switch result := readLine(&self.prompt); true {
		case result == nil:
			break L

		case *result != "":
			str += *result + "\n"

			self.setIndent()

			if indentCount <= 0 {
				if *result == "exit" {
					self.Stop()
					break L
				}

				hist := str[:len(str)-1]
				addHistory(hist) //allow user to recall this line
				self.history.WriteString(str)

				self.parseInput(str)

				str = ""
			}
		}
	}
}

func (self *JSRepl) PrintValue(v interface{}) {
	method, _ := self.re.Vm.Get("prettyPrint")
	v, err := self.re.Vm.ToValue(v)
	if err == nil {
		val, err := method.Call(method, v)
		if err == nil {
			fmt.Printf("%v", val)
		}
	}
}
