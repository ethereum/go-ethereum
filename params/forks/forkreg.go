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

package forks

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// Fork identifies a specific network upgrade (hard fork).
type Fork struct {
	*forkProperties
}

// BlockBased reports whether the fork is scheduled by block number.
// If false, it is assumed to be scheduled based on block timestamp.
func (d Fork) BlockBased() bool {
	return d.blockBased
}

// String returns the fork name.
func (d Fork) String() string {
	return d.name
}

// ConfigName returns the fork config name.
func (d Fork) ConfigName() string {
	return d.configName
}

// DirectDependencies iterates the fork's direct dependencies.
func (f Fork) DirectDependencies() iter.Seq[Fork] {
	return slices.Values(f.directDeps)
}

// Requires says whether a fork (transitively) depends on the fork given as parameter.
func (f Fork) Requires(other Fork) bool {
	_, ok := f.deps[other]
	return ok
}

func (f Fork) After(other Fork) bool {
	return f == other || f.Requires(other)
}

// UnmarshalText parses a fork name. Note this uses the config name.
func (f *Fork) UnmarshalText(v []byte) error {
	df, ok := ForkByConfigName(string(v))
	if !ok {
		return fmt.Errorf("unknown fork %q", v)
	}
	*f = df
	return nil
}

// MarshalText encodes the fork config name.
func (f *Fork) MarshalText() ([]byte, error) {
	return []byte(f.configName), nil
}

type forkProperties struct {
	name       string
	configName string
	blockBased bool
	directDeps []Fork
	deps       map[Fork]struct{}
}

// Spec is the definition of a fork.
type Spec struct {
	Name       string // the canonical name
	ConfigName string // the name used in genesis.json
	BlockBased bool   // whether scheduling is based on block number (false == scheduled by timestamp)
	Requires   []Fork // list of forks that must activate at or before this one
}

var (
	registry             = map[Fork]struct{}{}
	registryByName       = map[string]Fork{}
	registryByConfigName = map[string]Fork{}
)

// Define creates a fork definition in the registry.
// This is meant to be called at package initialization time.
func Define(ft Spec) Fork {
	if ft.Name == "" {
		panic("blank fork name")
	}
	if _, ok := registryByName[ft.Name]; ok {
		panic(fmt.Sprintf("fork %q already defined", ft.Name))
	}
	cname := ft.ConfigName
	if cname == "" {
		cname = strings.ToLower(ft.Name[:1]) + ft.Name[1:]
	}
	if _, ok := registryByConfigName[cname]; ok {
		panic(fmt.Sprintf("fork config name %q already defined", cname))
	}

	f := Fork{
		forkProperties: &forkProperties{
			name:       ft.Name,
			configName: cname,
			blockBased: ft.BlockBased,
			directDeps: slices.Clone(ft.Requires),
			deps:       make(map[Fork]struct{}),
		},
	}

	// Build the dependency set.
	for _, dep := range ft.Requires {
		if dep == f {
			panic("fork depends on itself")
		}
		if dep.Requires(f) {
			panic(fmt.Sprintf("fork dependency cycle: %v requires %v", dep, f))
		}
		maps.Copy(f.deps, dep.deps)
		f.deps[dep] = struct{}{}
	}

	// Add to registry.
	registry[f] = struct{}{}
	registryByName[f.name] = f
	registryByConfigName[cname] = f
	return f
}

// All iterates over defined forks in order of their names.
func All() iter.Seq[Fork] {
	sorted := slices.SortedFunc(maps.Keys(registry), func(f1, f2 Fork) int {
		return strings.Compare(f1.name, f2.name)
	})
	return slices.Values(sorted)
}

// ForkByName returns a fork by its canonical name.
func ForkByName(name string) (Fork, bool) {
	f, ok := registryByName[name]
	return f, ok
}

// ForkByConfigName returns a fork by its configuration name.
func ForkByConfigName(name string) (Fork, bool) {
	f, ok := registryByConfigName[name]
	return f, ok
}
