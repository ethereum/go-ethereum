// Copyright 2015 The go-ethereum Authors
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

package backends

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

type SimulatedBackend struct {
	simulated.Backend
}

// New creates a new binding backend using a simulated blockchain
// for testing purposes.
//
// A simulated backend always uses chainID 1337.
//
// Deprecated: please use simulated.Backend from package
// github.com/ethereum/go-ethereum/ethclient/simulated instead.
func NewSimulatedBackend(alloc core.GenesisAlloc, gasLimit uint64) SimulatedBackend {
	return SimulatedBackend{simulated.New(alloc, gasLimit)}
}
