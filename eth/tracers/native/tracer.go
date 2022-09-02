// Copyright 2021 The go-ethereum Authors
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

/*
Package native is a collection of tracers written in go.

In order to add a native tracer and have it compiled into the binary, a new
file needs to be added to this folder, containing an implementation of the
`eth.tracers.Tracer` interface.

Aside from implementing the tracer, it also needs to register itself, using the
`register` method -- and this needs to be done in the package initialization.

Example:

```golang

	func init() {
		register("noopTracerNative", newNoopTracer)
	}

```
*/
package native

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/eth/tracers"
)

// init registers itself this packages as a lookup for tracers.
func init() {
	tracers.RegisterLookup(false, lookup)
}

// ctorFn is the constructor signature of a native tracer.
type ctorFn = func(*tracers.Context, json.RawMessage) (tracers.Tracer, error)

/*
ctors is a map of package-local tracer constructors.

We cannot be certain about the order of init-functions within a package,
The go spec (https://golang.org/ref/spec#Package_initialization) says

> To ensure reproducible initialization behavior, build systems
> are encouraged to present multiple files belonging to the same
> package in lexical file name order to a compiler.

Hence, we cannot make the map in init, but must make it upon first use.
*/
var ctors map[string]ctorFn

// register is used by native tracers to register their presence.
func register(name string, ctor ctorFn) {
	if ctors == nil {
		ctors = make(map[string]ctorFn)
	}
	ctors[name] = ctor
}

// lookup returns a tracer, if one can be matched to the given name.
func lookup(name string, ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	if ctors == nil {
		ctors = make(map[string]ctorFn)
	}
	if ctor, ok := ctors[name]; ok {
		return ctor(ctx, cfg)
	}
	return nil, errors.New("no tracer found")
}
