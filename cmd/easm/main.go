package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	app *cli.App

	DebugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "outputs lexer and compiler debug output",
	}
)

func init() {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	app.Email = ""
	app.Version = "0.1"

	app.Flags = []cli.Flag{
		DebugFlag,
	}
	app.Action = run
}

func run(ctx *cli.Context) {
	debug := ctx.GlobalBool(DebugFlag.Name)

	if len(ctx.Args()) == 0 {
		glog.Exitln("err: <filename> required")
	}

	fn := ctx.Args().First()
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		glog.Exitln("err:", err)
	}

	compiler := newCompiler(debug)
	compiler.feed(lex(fn, src, debug))

	bin, errors := compiler.compile()
	if len(errors) > 0 {
		// report errors
		for _, err := range errors {
			got, want := err.Error()
			fmt.Printf("%s:%d: syntax error: unexpected %v, expected %v\n", fn, err.Lineno(), got, want)
		}
		os.Exit(1)
	}
	fmt.Println(bin)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Compiler contains information about the parsed source
// and holds the tokens for the program.
type Compiler struct {
	tokens []item
	binary []interface{}

	labels map[string]int

	pc, pos int

	debug bool
}

// newCompiler returns a new allocated compiler.
func newCompiler(debug bool) *Compiler {
	return &Compiler{
		labels: make(map[string]int),
		debug:  debug,
	}
}

// feed feeds tokens in to ch and are interpreted by
// the compiler.
//
// feed is the first pass in the compile stage as it
// collect the used labels in the program and keeps a
// program counter which is used to determine the locations
// of the jump dests. The labels can than be used in the
// second stage to push labels and determine the right
// position.
func (c *Compiler) feed(ch <-chan item) {
	for i := range ch {
		switch i.typ {
		case number:
			num := common.String2Big(i.text).Bytes()
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
		fmt.Println("found", len(c.labels), "labels")
	}
}

// compile compiles the current tokens and returns a
// binary string that can be interpreted by the EVM
// and an error if it failed.
//
// compile is the second stage in the compile phase
// which compiles the tokens to EVM instructions.
func (c *Compiler) compile() (string, []cerror) {
	var errors []cerror
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
// posititon.
func (c *Compiler) next() item {
	token := c.tokens[c.pos]
	c.pos++
	return token
}

// compile line compiles a single line instruction e.g.
// "push 1", "jump @labal".
func (c *Compiler) compileLine() cerror {
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
func (c *Compiler) compileNumber(element item) (int, error) {
	num := common.String2Big(element.text).Bytes()
	if len(num) == 0 {
		num = []byte{0}
	}
	c.pushBin(num)
	return len(num), nil
}

// compileElement compiles the element (push & label or both)
// to a binary representation and may error if incorrect statements
// where fed.
func (c *Compiler) compileElement(element item) cerror {
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
	}
	// push the operation
	c.pushBin(toBinary(element.text))

	// handle pushes. pushes are read from left to right.
	if isPush(element.text) {
		rvalue := c.next()
		switch rvalue.typ {
		case number:
			// TODO figure out how to return the error properly
			c.compileNumber(rvalue)
		case stringValue:
			// strings are quoted, remove them.
			c.pushBin(rvalue.text[1 : len(rvalue.text)-1])
		case label:
			c.pushBin(vm.PUSH4)
			pos := make([]byte, 4)
			copy(pos, big.NewInt(int64(c.labels[rvalue.text])).Bytes())
			c.pushBin(pos)
		default:
			return compileErr(rvalue, rvalue.text, "number, string or label")
		}
	}

	return nil
}

// compileLabel pushes a jumpdest to the binary slice.
func (c *Compiler) compileLabel() cerror {
	c.pushBin(vm.JUMPDEST)
	return nil
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
	if op == "push" {
		return true
	}
	return false
}

// isJump returns whether the string op is jump(i)
func isJump(op string) bool {
	return op == "jumpi" || op == "jump"
}

// toBinary converts text to a vm.OpCode
func toBinary(text string) vm.OpCode {
	if isPush(text) {
		text = "push1"
	}
	return vm.StringToOp(strings.ToUpper(text))
}

type cerror interface {
	Error() (string, string)
	Lineno() int
}

type compileError struct {
	got  string
	want string

	lineno int
}

func (c compileError) Error() (string, string) { return c.got, c.want }
func (c compileError) Lineno() int             { return c.lineno }

var (
	errExpBol            = errors.New("expected beginning of line")
	errExpElementOrLabel = errors.New("expected beginning of line")
)

func compileErr(c item, got, want string) cerror {
	return compileError{
		got:    got,
		want:   want,
		lineno: c.lineno,
	}
}
