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

// all contains all the built in JavaScript tracers by name.
var all = make(map[string]string)

// init retrieves the JavaScript transaction tracers included in go-ethereum.
func init() {
	for _, file := range tracers.AssetNames() {
		// Convert the underscored tracer file name into a camelcase tracer name
		pieces := strings.Split(strings.TrimSuffix(file, ".js"), "_")
		for i := 1; i < len(pieces); i++ {
			pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
		}
		name := strings.Join(pieces, "")

		// Retrieve and store the tracer
		all[name] = string(tracers.MustAsset(file))
	}
}

// Tracer retrieves a specific JavaScript tracer by name.
func Tracer(name string) (string, bool) {
	if tracer, ok := all[name]; ok {
		return tracer, true
	}
	return "", false
}
