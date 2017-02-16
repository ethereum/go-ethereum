package main

import "fmt"

func main() {
	program := `
	jump @main
my_label:
	push 0
	push "hello"
	mstore

	push 0
	push 5
	push 10
	log

main:
	push 1
	push 1
	eq

	jumpif @my_label
`
	ch := lex("program.asm", program)
	for i := range ch {
		fmt.Printf("%04d: (%-20v) %s\n", i.lineno, i.typ, i.text)
	}
}
