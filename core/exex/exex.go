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

// Package exex contains the stable API of the Geth Execution Extensions.
package exex

import (
	"github.com/ethereum/go-ethereum/log"
)

// RegisterV1 registers an execution extension plugin with a unique name.
func RegisterV1(name string, constructor NewPluginV1) {
	globalRegistry.RegisterV1(name, constructor)
}

// NewPluginV1 is the constructor signature for making a new plugin.
type NewPluginV1 func(config *ConfigV1) (*PluginV1, error)

// ConfigV1 contains some configurations for initializing exex plugins. Some of
// the fields originate from Geth, other fields from user configs.
type ConfigV1 struct {
	Logger log.Logger // Geth's logger with the plugin name injected

	User string // Opaque flag provided by the user on the CLI
}

// PluginV1 is an Execution Extension module that can be injected into Geth's
// processing pipeline to subscribe to different node, chain and EVM lifecycle
// events.
//
// Note, V1 of the Execution Extension plugin module has not yet been stabilized.
// There might be breaking changes until it is tagged as released!
type PluginV1 struct {
	OnInit  InitHook  // Called when the chain gets initialized within Geth
	OnClose CloseHook // Called when the chain gets torn down within Geth
	OnHead  HeadHook  // Called when the chain head block is updated in Geth
	OnReorg ReorgHook // Called wnen the chain reorgs to a sidechain within Geth
	OnFinal FinalHook // Called when the chain finalizes a block within Geth
}
