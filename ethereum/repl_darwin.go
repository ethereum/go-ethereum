package main

// #cgo LDFLAGS: -lreadline
// #include <stdio.h>
// #include <stdlib.h>
// #include <readline/readline.h>
// #include <readline/history.h>
import "C"

import (
	"strings"
	"unsafe"
)

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
L:
	for {
		switch result := readLine(&self.prompt); true {
		case result == nil:
			break L //exit loop

		case *result != "": //ignore blank lines
			str += *result + "\n"

			self.setIndent()

			if indentCount <= 0 {
				addHistory(str) //allow user to recall this line

				self.parseInput(str)
			}
		}
	}
}
