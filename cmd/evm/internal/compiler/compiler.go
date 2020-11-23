// Copyright 2017 The go-ethereum Authors
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

package compiler

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/asm"
)

func Compile(fn string, src []byte, debug bool) (string, error) {
	compiler := asm.NewCompiler(debug)
	compiler.Feed(asm.Lex(fn, src, debug))

	bin, compileErrors := compiler.Compile()
	if len(compileErrors) > 0 {
		// report errors
		errs := ""
		for _, err := range compileErrors {
			errs += fmt.Sprintf("%s:%v\n", fn, err)
		}
		return "", errors.New(errs + "compiling failed\n")
	}
	return bin, nil
}
