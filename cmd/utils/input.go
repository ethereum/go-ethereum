// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"fmt"
	"strings"

	"github.com/peterh/liner"
)

// Holds the stdin line reader.
// Only this reader may be used for input because it keeps
// an internal buffer.
var Stdin = newUserInputReader()

type userInputReader struct {
	*liner.State
	warned     bool
	supported  bool
	normalMode liner.ModeApplier
	rawMode    liner.ModeApplier
}

func newUserInputReader() *userInputReader {
	r := new(userInputReader)
	// Get the original mode before calling NewLiner.
	// This is usually regular "cooked" mode where characters echo.
	normalMode, _ := liner.TerminalMode()
	// Turn on liner. It switches to raw mode.
	r.State = liner.NewLiner()
	rawMode, err := liner.TerminalMode()
	if err != nil || !liner.TerminalSupported() {
		r.supported = false
	} else {
		r.supported = true
		r.normalMode = normalMode
		r.rawMode = rawMode
		// Switch back to normal mode while we're not prompting.
		normalMode.ApplyMode()
	}
	return r
}

func (r *userInputReader) Prompt(prompt string) (string, error) {
	if r.supported {
		r.rawMode.ApplyMode()
		defer r.normalMode.ApplyMode()
	} else {
		// liner tries to be smart about printing the prompt
		// and doesn't print anything if input is redirected.
		// Un-smart it by printing the prompt always.
		fmt.Print(prompt)
		prompt = ""
		defer fmt.Println()
	}
	return r.State.Prompt(prompt)
}

func (r *userInputReader) PasswordPrompt(prompt string) (passwd string, err error) {
	if r.supported {
		r.rawMode.ApplyMode()
		defer r.normalMode.ApplyMode()
		return r.State.PasswordPrompt(prompt)
	}
	if !r.warned {
		fmt.Println("!! Unsupported terminal, password will be echoed.")
		r.warned = true
	}
	// Just as in Prompt, handle printing the prompt here instead of relying on liner.
	fmt.Print(prompt)
	passwd, err = r.State.Prompt("")
	fmt.Println()
	return passwd, err
}

func (r *userInputReader) ConfirmPrompt(prompt string) (bool, error) {
	prompt = prompt + " [y/N] "
	input, err := r.Prompt(prompt)
	if len(input) > 0 && strings.ToUpper(input[:1]) == "Y" {
		return true, nil
	}
	return false, err
}
