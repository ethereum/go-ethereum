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

package exex

// globalRegistry is the public plugin registry to inject execution extensions into.
var globalRegistry = newRegistry()

// Registry retrieves the global plugin registry.
//
// Note, the downcast to interface{} is deliberate to hide all the methods on the
// registry and avoid plugins from accidentally poking at unintended internals.
func Registry() interface{} {
	return globalRegistry
}

// registry is the collection of Execution Extension plugins which can be used
// to extend Geth's functionality with external code.
type registry struct {
	pluginsMakersV1 map[string]NewPluginV1
	pluginsV1       map[string]*PluginV1
}

// newRegistry creates a new exex plugin registry.
func newRegistry() *registry {
	return &registry{
		pluginsMakersV1: make(map[string]NewPluginV1),
		pluginsV1:       make(map[string]*PluginV1),
	}
}

// RegisterV1 registers an execution extension plugin with a unique name.
func (reg *registry) RegisterV1(name string, constructor NewPluginV1) {
	reg.pluginsMakersV1[name] = constructor
}
