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
	"encoding/json"
	"strings"
	"unicode"

	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers/internal/tracers"
)

// Tracer interface extends vm.EVMLogger and additionally
// allows collecting the tracing result.
type Tracer interface {
	vm.EVMLogger
	GetResult() (json.RawMessage, error)
	// Stop terminates execution of the tracer at the first opportune moment.
	Stop(err error)
}

var (
	nativeTracers map[string]func() Tracer = make(map[string]func() Tracer)
	jsTracers                              = make(map[string]string)
)

// RegisterNativeTracer makes native tracers which adhere
// to the `Tracer` interface available to the rest of the codebase.
// It is typically invoked in the `init()` function, e.g. see the `native/call.go`.
func RegisterNativeTracer(name string, ctor func() Tracer) {
	nativeTracers[name] = ctor
}

// New returns a new instance of a tracer,
//  1. If 'code' is the name of a registered native tracer, then that tracer
//     is instantiated and returned
//  2. If 'code' is the name of a registered js-tracer, then that tracer is
//     instantiated and returned
//  3. Otherwise, the code is interpreted as the js code of a js-tracer, and
//     is evaluated and returned.
func New(code string, ctx *Context) (Tracer, error) {
	// Resolve native tracer
	if fn, ok := nativeTracers[code]; ok {
		return fn(), nil
	}
	// Resolve js-tracers by name and assemble the tracer object
	if tracer, ok := jsTracers[code]; ok {
		code = tracer
	}
	return NewJsTracer(code, ctx)
}

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
		name := camel(strings.TrimSuffix(file, ".js"))
		jsTracers[name] = string(tracers.MustAsset(file))

	}
}
