// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package asm

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Compiler contains information about the parsed source
// and holds the tokens for the program.
type Compiler struct {
	tokens []token
	binary []interface{}

	labels map[string]int

	pc, pos int

	debug bool
}

// newCompiler returns a new allocated compiler.
func NewCompiler(debug bool) *Compiler {
	return &Compiler{
		labels: make(map[string]int),
		debug:  debug,
	}
}

// Feed feeds tokens in to ch and are interpreted by
// the compiler.
//
// feed is the first pass in the compile stage as it
// collect the used labels in the program and keeps a
// program counter which is used to determine the locations
// of the jump dests. The labels can than be used in the
// second stage to push labels and determine the right
// position.
func (c *Compiler) Feed(ch <-chan token) {
	for i := range ch {
		switch i.typ {
		case number:
			num := math.MustParseBig256(i.text).Bytes()
			if len(num) == 0 {
				num = []byte{0}
			}
			c.pc += len(num)
		case stringValue:
			c.pc += len(i.text) - 2
		case element:
			c.pc++
		case labelDef:
			c.labels[i.text] = c.pc
			c.pc++
		case label:
			c.pc += 5
		}

		c.tokens = append(c.tokens, i)
	}
	if c.debug {
		fmt.Fprintln(os.Stderr, "found", len(c.labels), "labels")
	}
}

// Compile compiles the current tokens and returns a
// binary string that can be interpreted by the EVM
// and an error if it failed.
//
// compile is the second stage in the compile phase
// which compiles the tokens to EVM instructions.
func (c *Compiler) Compile() (string, []error) {
	var errors []error
	// continue looping over the tokens until
	// the stack has been exhausted.
	for c.pos < len(c.tokens) {
		if err := c.compileLine(); err != nil {
			errors = append(errors, err)
		}
	}

	// turn the binary to hex
	var bin string
	for _, v := range c.binary {
		switch v := v.(type) {
		case vm.OpCode:
			bin += fmt.Sprintf("%x", []byte{byte(v)})
		case []byte:
			bin += fmt.Sprintf("%x", v)
		}
	}
	return bin, errors
}

// next returns the next token and increments the
// position.
func (c *Compiler) next() token {
	token := c.tokens[c.pos]
	c.pos++
	return token
}

// compile line compiles a single line instruction e.g.
// "push 1", "jump @label".
func (c *Compiler) compileLine() error {
	n := c.next()
	if n.typ != lineStart {
		return compileErr(n, n.typ.String(), lineStart.String())
	}

	lvalue := c.next()
	switch lvalue.typ {
	case eof:
		return nil
	case element:
		if err := c.compileElement(lvalue); err != nil {
			return err
		}
	case labelDef:
		c.compileLabel()
	case lineEnd:
		return nil
	default:
		return compileErr(lvalue, lvalue.text, fmt.Sprintf("%v or %v", labelDef, element))
	}

	if n := c.next(); n.typ != lineEnd {
		return compileErr(n, n.text, lineEnd.String())
	}

	return nil
}

// compileNumber compiles the number to bytes
func (c *Compiler) compileNumber(element token) (int, error) {
	num := math.MustParseBig256(element.text).Bytes()
	if len(num) == 0 {
		num = []byte{0}
	}
	c.pushBin(num)
	return len(num), nil
}

// compileElement compiles the element (push & label or both)
// to a binary representation and may error if incorrect statements
// where fed.
func (c *Compiler) compileElement(element token) error {
	// check for a jump. jumps must be read and compiled
	// from right to left.
	if isJump(element.text) {
		rvalue := c.next()
		switch rvalue.typ {
		case number:
			// TODO figure out how to return the error properly
			c.compileNumber(rvalue)
		case stringValue:
			// strings are quoted, remove them.
			c.pushBin(rvalue.text[1 : len(rvalue.text)-2])
		case label:
			c.pushBin(vm.PUSH4)
			pos := big.NewInt(int64(c.labels[rvalue.text])).Bytes()
			pos = append(make([]byte, 4-len(pos)), pos...)
			c.pushBin(pos)
		default:
			return compileErr(rvalue, rvalue.text, "number, string or label")
		}
		// push the operation
		c.pushBin(toBinary(element.text))
		return nil
	} else if isPush(element.text) {
		// handle pushes. pushes are read from left to right.
		var value []byte

		rvalue := c.next()
		switch rvalue.typ {
		case number:
			value = math.MustParseBig256(rvalue.text).Bytes()
			if len(value) == 0 {
				value = []byte{0}
			}
		case stringValue:
			value = []byte(rvalue.text[1 : len(rvalue.text)-1])
		case label:
			value = make([]byte, 4)
			copy(value, big.NewInt(int64(c.labels[rvalue.text])).Bytes())
		default:
			return compileErr(rvalue, rvalue.text, "number, string or label")
		}

		if len(value) > 32 {
			return fmt.Errorf("%d type error: unsupported string or number with size > 32", rvalue.lineno)
		}

		c.pushBin(vm.OpCode(int(vm.PUSH1) - 1 + len(value)))
		c.pushBin(value)
	} else {
		c.pushBin(toBinary(element.text))
	}

	return nil
}

// compileLabel pushes a jumpdest to the binary slice.
func (c *Compiler) compileLabel() {
	c.pushBin(vm.JUMPDEST)
}

// pushBin pushes the value v to the binary stack.
func (c *Compiler) pushBin(v interface{}) {
	if c.debug {
		fmt.Printf("%d: %v\n", len(c.binary), v)
	}
	c.binary = append(c.binary, v)
}

// isPush returns whether the string op is either any of
// push(N).
func isPush(op string) bool {
	return op == "push"
}

// isJump returns whether the string op is jump(i)
func isJump(op string) bool {
	return op == "jumpi" || op == "jump"
}

// toBinary converts text to a vm.OpCode
func toBinary(text string) vm.OpCode {
	return vm.StringToOp(strings.ToUpper(text))
}

type compileError struct {
	got  string
	want string

	lineno int
}

func (err compileError) Error() string {
	return fmt.Sprintf("%d syntax error: unexpected %v, expected %v", err.lineno, err.got, err.want)
}

func compileErr(c token, got, want string) error {
	return compileError{
		got:    got,
		want:   want,
		lineno: c.lineno,
	}
}
