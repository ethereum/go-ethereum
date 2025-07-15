// Copyright 2025 The go-ethereum Authors
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

package params

import (
	"fmt"
	"reflect"
)

type ParameterType any

type paramInfo struct {
	rtype      reflect.Type
	name       string
	optional   bool
	defaultVal any
}

var (
	paramRegistry = map[reflect.Type]*paramInfo{}
	paramRegistryByName = map[string]*paramInfo{}
)

type Parameter[T ParameterType] struct {
	Name     string
	Optional bool
	Default  T
}

// Get retrieves the value of a chain parameter.
func Get[T ParameterType](cfg *Config2) T {
	for _, p := range cfg.param {
		if v, ok := p.(T); ok {
			return v
		}
	}
	// get default
	var z T
	info, ok := findParam(z)
	if !ok {
		panic(fmt.Sprintf("unknown parameter type %T", z))
	}
	return info.defaultVal.(T)
}

func Define[T ParameterType](def Parameter[T]) {
	var z T
	info, defined := paramRegistryByName[def.Name]
	if defined {
		panic(fmt.Sprintf("chain parameter %q already registered with type %v", info.name, info.rtype))
	}
	rtype := reflect.TypeOf(z)
	info, defined = paramRegistry[rtype]
	if defined {
		panic(fmt.Sprintf("chain parameter of type %v already registered with name %q", rtype, info.name))
	}
	info = &paramInfo{
		rtype:      rtype,
		name:       def.Name,
		optional:   def.Optional,
		defaultVal: def.Default,
	}
	paramRegistry[rtype] = info
	paramRegistryByName[def.Name] = info
}

func findParam(v any) (*paramInfo, bool) {
	rtype := reflect.TypeOf(v)
	info, ok := paramRegistry[rtype]
	return info, ok
}
