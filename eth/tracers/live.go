// Copyright 2024 The go-ethereum Authors
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

package tracers

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/core/tracing"
)

type ctorFunc func(config json.RawMessage) (*tracing.Hooks, error)

// LiveDirectory is the collection of tracers which can be used
// during normal block import operations.
var LiveDirectory = liveDirectory{elems: make(map[string]ctorFunc)}

type liveDirectory struct {
	elems map[string]ctorFunc
}

// Register registers a tracer constructor by name.
func (d *liveDirectory) Register(name string, f ctorFunc) {
	d.elems[name] = f
}

// New instantiates a tracer by name.
func (d *liveDirectory) New(name string, config json.RawMessage) (*tracing.Hooks, error) {
	if len(config) == 0 {
		config = json.RawMessage("{}")
	}
	if f, ok := d.elems[name]; ok {
		return f(config)
	}
	return nil, errors.New("not found")
}
