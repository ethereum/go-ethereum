// Copyright 2023 The go-ethereum Authors
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

package misc

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ApplyBeaconRoot adds the beacon root from the header to the state.
func ApplyBeaconRoot(header *types.Header, state *state.StateDB) {
	// If EIP-4788 is enabled, we need to store the block root
	historicalStorageAddress := common.BytesToAddress([]byte{20})
	timeIndex := header.Time % params.HistoricalRootModulus
	rootIndex := timeIndex + params.HistoricalRootModulus
	time := header.Time
	// timeIndex -> header.Time
	state.SetState(historicalStorageAddress, common.Uint64ToHash(timeIndex), common.Uint64ToHash(time))
	// rootIndex -> header.BeaconRoot
	state.SetState(historicalStorageAddress, common.Uint64ToHash(rootIndex), *header.BeaconRoot)
}
