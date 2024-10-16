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

import (
	"github.com/ethereum/go-ethereum/core/exex"
	"github.com/ethereum/go-ethereum/core/types"
)

// globalRegistry is the Geth internal version of the exex registry with the
// trigger methods exposed to be callable from within Geth.
var globalRegistry registry

func init() {
	globalRegistry = exex.Registry().(registry)
}

// registry exposes all the hidden methods on the plugin registry to allow event
// triggers to be invoked.
type registry interface {
	Plugins() []string
	Instantiate(name string, userconf string) error

	TriggerInitHook(chain exex.Chain)
	TriggerCloseHook()
	TriggerHeadHook(head *types.Header)
}

// Plugins returns a list of all registered plugins to generate CLI flags.
func Plugins() []string {
	return globalRegistry.Plugins()
}

// Instantiate constructs an execution extension plugin from a unique name.
func Instantiate(name string, userconf string) error {
	return globalRegistry.Instantiate(name, userconf)
}

// TriggerInitHook triggers the OnInit hook in exex plugins.
func TriggerInitHook(chain gethChain) {
	globalRegistry.TriggerInitHook(wrapChain(chain))
}

// TriggerCloseHook triggers the OnClose hook in exex plugins.
func TriggerCloseHook() {
	globalRegistry.TriggerCloseHook()
}

// TriggerHeadHook triggers the OnHead hook in exex plugins.
func TriggerHeadHook(head *types.Header) {
	globalRegistry.TriggerHeadHook(head)
}
