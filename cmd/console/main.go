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
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

const (
	ClientIdentifier = "Geth console"
	Version          = "0.9.26"
)

func main() {
	// Wrap the standard output with a colorified stream (windows)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if pr, pw, err := os.Pipe(); err == nil {
			go io.Copy(colorable.NewColorableStdout(), pr)
			os.Stdout = pw
		}
	}

	// TODO, datadir + jspath
	repl := newJSRE("/home/bas/.ethereum", ".")
	repl.interactive()
}
