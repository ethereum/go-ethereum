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

package plugins

import (
	"github.com/ethereum/go-ethereum/core/exex"
	"github.com/ethereum/go-ethereum/core/types"
)

// Register the minimal ExEx plugin into Geth.
func init() {
	exex.RegisterV1("minimal", newMinimalPlugin)
}

// newMinimalPlugin creates a minimal Execution Extension plugin to react to some
// chain events.
func newMinimalPlugin(config *exex.ConfigV1) (*exex.PluginV1, error) {
	return &exex.PluginV1{
		OnHead: func(head *types.Header) {
			config.Logger.Info("Chain head updated", "number", head.Number, "hash", head.Hash())
		},
	}, nil
}
