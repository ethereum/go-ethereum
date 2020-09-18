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

// Package tracers is a collection of JavaScript transaction tracers.
package tracers

import (
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/eth/tracers/internal/tracers"
)

// allJs contains allJs the built in JavaScript tracers by name.
var allJs = make(map[string]string)
var allWasm = make(map[string]string)

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}

// init retrieves the JavaScript transaction tracers included in go-ethereum.
func init() {
	for _, file := range tracers.AssetNames() {
		if strings.HasSuffix(file, ".js") {
			name := camel(strings.TrimSuffix(file, ".js"))
			allJs[name] = string(tracers.MustAsset(file))
		} else {
			name := camel(strings.TrimSuffix(file, ".wasm"))
			allWasm[name] = string(tracers.MustAsset(file))
		}
	}
}

// tracer retrieves a specific JavaScript or Wasm tracer by name.
func tracer(name string) (string, bool, bool) {
	if tracer, ok := allJs[name]; ok {
		return tracer, false, true
	}
	if tracer, ok := allWasm[name]; ok {
		return tracer, true, true
	}
	return "", false, false
}
