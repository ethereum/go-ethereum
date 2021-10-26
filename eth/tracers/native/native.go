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

package native

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Tracer interface extends vm.Tracer and additionally
// allows collecting the tracing result.
type Tracer interface {
	vm.Tracer
	GetResult() (json.RawMessage, error)
}

// constructor creates a new instance of a Tracer.
type constructor func() Tracer

var tracers map[string]constructor = make(map[string]constructor)

// register makes native tracers in this directory which adhere
// to the `Tracer` interface available to the rest of the codebase.
// It is typically invoked in the `init()` function.
func register(name string, fn constructor) {
	tracers[name] = fn
}

// New returns a new instance of a tracer, if one was
// registered under the given name.
func New(name string) (Tracer, bool) {
	if fn, ok := tracers[name]; ok {
		return fn(), true
	}
	return nil, false
}
