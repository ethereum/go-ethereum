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
)

type Parameter[V any] struct{
	info regInfo
}

func (p Parameter[V]) Get(cfg *Config2) V {
	v, ok := cfg.param[p.info.id]
	if ok {
		return v.(V)
	}
	return p.info.defaultValue.(V)
}

func (p Parameter[V]) V(v V) ParamValue {
	return ParamValue{p.info.id, v}
}

type ParamValue struct{
	id int
	value any
}

var (
	paramCounter int
	paramRegistry = map[int]regInfo{}
	paramRegistryByName = map[string]int{}
)

type T[V any] struct{
	Name string
	Optional bool
	Default V
	Validate func(v V, cfg *Config2) error
}

type regInfo struct{
	id int
	name string
	optional bool
	defaultValue any
	new func() any
	validate func(any, *Config2) error
}

// Define creates a chain parameter in the registry.
func Define[V any](def T[V]) Parameter[V] {
	if id, defined := paramRegistryByName[def.Name]; defined {
		info := paramRegistry[id]
		panic(fmt.Sprintf("chain parameter %q already registered with type %T", def.Name, info.defaultValue))
	}

	id := paramCounter
	paramCounter++

	regInfo := regInfo{
		id: id,
		name: def.Name,
		optional: def.Optional,
		defaultValue: def.Default,
		new: func() any {
			var z V
			return z
		},
		validate: func(v any, cfg *Config2) error {
			if def.Validate == nil {
				return nil
			}
			return def.Validate(v.(V), cfg)
		},
	}
	paramRegistry[id] = regInfo
	paramRegistryByName[def.Name] = id
	return Parameter[V]{info: regInfo}
}
