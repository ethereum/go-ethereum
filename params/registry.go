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
	"strings"
)

// Parameter represents a chain parameter.
// Parameters are globally registered using `Define`.
type Parameter[V any] struct {
	info regInfo
}

// Get retrieves the value of a parameter from a config.
func (p Parameter[V]) Get(cfg *Config2) V {
	if p.info.id == 0 {
		panic("zero parameter")
	}
	v, ok := cfg.param[p.info.id]
	if ok {
		return v.(V)
	}
	return p.info.defaultValue.(V)
}

// Defined reports whether the parameter is set in the config.
func (p Parameter[V]) Defined(cfg *Config2) bool {
	if p.info.id == 0 {
		panic("zero parameter")
	}
	_, ok := cfg.param[p.info.id]
	return ok
}

// V creates a ParamValue with the given value. You need this to
// specify parameter values when constructing a Config in code.
func (p Parameter[V]) V(v V) ParamValue {
	if p.info.id == 0 {
		panic("zero parameter")
	}
	return ParamValue{p.info.id, v}
}

// ParamValue contains a chain parameter and its value.
// This is created by calling `V` on the parameter.
type ParamValue struct {
	id    int
	value any
}

var (
	paramCounter        = int(1)
	paramRegistry       = map[int]regInfo{}
	paramRegistryByName = map[string]int{}
)

// T is the definition of a chain parameter type.
type T[V any] struct {
	Name     string // the parameter name
	Optional bool   // optional says
	Default  V
	Validate func(v V, cfg *Config2) error
}

type regInfo struct {
	id           int
	name         string
	optional     bool
	defaultValue any
	new          func() any
	validate     func(any, *Config2) error
}

// Define creates a chain parameter in the registry.
// This is meant to be called at package initialization time.
func Define[V any](def T[V]) Parameter[V] {
	if def.Name == "" {
		panic("blank parameter name")
	}
	if id, defined := paramRegistryByName[def.Name]; defined {
		info := paramRegistry[id]
		panic(fmt.Sprintf("chain parameter %q already registered with type %T", def.Name, info.defaultValue))
	}
	if strings.HasSuffix(def.Name, "Block") || strings.HasSuffix(def.Name, "Time") {
		panic("chain parameter name cannot end in 'Block' or 'Time'")
	}

	id := paramCounter
	paramCounter++

	regInfo := regInfo{
		id:           id,
		name:         def.Name,
		optional:     def.Optional,
		defaultValue: def.Default,
		new: func() any {
			return new(V)
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
