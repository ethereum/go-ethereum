package main

// #cgo LDFLAGS: -lreadline
// #include <stdio.h>
// #include <stdlib.h>
// #include <readline/readline.h>
// #include <readline/history.h>
import "C"
import "unsafe"

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

func (self *JSRepl) read() {
	prompt := "eth >>> "

L:
	for {
		switch result := readLine(&prompt); true {
		case result == nil:
			break L //exit loop

		case *result != "": //ignore blank lines
			addHistory(*result) //allow user to recall this line

			self.parseInput(*result)
		}
	}
}
