// Package repl implements a REPL (read-eval-print loop) for otto.
package repl

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/robertkrimen/otto"
	"gopkg.in/readline.v1"
)

var counter uint32

// DebuggerHandler implements otto's debugger handler signature, providing a
// simple drop-in debugger implementation.
func DebuggerHandler(vm *otto.Otto) {
	i := atomic.AddUint32(&counter, 1)

	// purposefully ignoring the error here - we can't do anything useful with
	// it except panicking, and that'd be pretty rude. it'd be easy enough for a
	// consumer to define an equivalent function that _does_ panic if desired.
	_ = RunWithPrompt(vm, fmt.Sprintf("DEBUGGER[%d]>", i))
}

// Run creates a REPL with the default prompt and no prelude.
func Run(vm *otto.Otto) error {
	return RunWithPromptAndPrelude(vm, "", "")
}

// RunWithPrompt runs a REPL with the given prompt and no prelude.
func RunWithPrompt(vm *otto.Otto, prompt string) error {
	return RunWithPromptAndPrelude(vm, prompt, "")
}

// RunWithPrelude runs a REPL with the default prompt and the given prelude.
func RunWithPrelude(vm *otto.Otto, prelude string) error {
	return RunWithPromptAndPrelude(vm, "", prelude)
}

// RunWithPromptAndPrelude runs a REPL with the given prompt and prelude.
func RunWithPromptAndPrelude(vm *otto.Otto, prompt, prelude string) error {
	if prompt == "" {
		prompt = ">"
	}

	prompt = strings.Trim(prompt, " ")
	prompt += " "

	rl, err := readline.New(prompt)
	if err != nil {
		return err
	}

	if prelude != "" {
		if _, err := io.Copy(rl.Stderr(), strings.NewReader(prelude+"\n")); err != nil {
			return err
		}

		rl.Refresh()
	}

	var d []string

	for {
		l, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if d != nil {
					d = nil

					rl.SetPrompt(prompt)
					rl.Refresh()

					continue
				}

				break
			}

			return err
		}

		if l == "" {
			continue
		}

		d = append(d, l)

		s, err := vm.Compile("repl", strings.Join(d, "\n"))
		if err != nil {
			rl.SetPrompt(strings.Repeat(" ", len(prompt)))
		} else {
			rl.SetPrompt(prompt)

			d = nil

			v, err := vm.Eval(s)
			if err != nil {
				if oerr, ok := err.(*otto.Error); ok {
					io.Copy(rl.Stdout(), strings.NewReader(oerr.String()))
				} else {
					io.Copy(rl.Stdout(), strings.NewReader(err.Error()))
				}
			} else {
				rl.Stdout().Write([]byte(v.String() + "\n"))
			}
		}

		rl.Refresh()
	}

	return rl.Close()
}
