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

package filtermaps

import (
	_ "embed"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

// checkpointList lists checkpoints for finalized epochs of a given chain.
// This allows the indexer to start indexing from the latest available
// checkpoint and then index tail epochs in reverse order.
type checkpointList []epochCheckpoint

// epochCheckpoint specified the last block of the epoch and the first log
// value index where that block starts. This allows a log value iterator to
// be initialized at the epoch boundary.
type epochCheckpoint struct {
	BlockNumber uint64 // block that generated the last log value of the given epoch
	BlockId     common.Hash
	FirstIndex  uint64 // first log value index of the given block
}

//go:embed checkpoints_mainnet.json
var checkpointsMainnetJSON []byte

//go:embed checkpoints_sepolia.json
var checkpointsSepoliaJSON []byte

//go:embed checkpoints_holesky.json
var checkpointsHoleskyJSON []byte

//go:embed checkpoints_hoodi.json
var checkpointsHoodiJSON []byte

// checkpoints lists sets of checkpoints for multiple chains. The matching
// checkpoint set is autodetected by the indexer once the canonical chain is
// known.
var checkpoints = []checkpointList{
	decodeCheckpoints(checkpointsMainnetJSON),
	decodeCheckpoints(checkpointsSepoliaJSON),
	decodeCheckpoints(checkpointsHoleskyJSON),
	decodeCheckpoints(checkpointsHoodiJSON),
}

func decodeCheckpoints(encoded []byte) (result checkpointList) {
	if err := json.Unmarshal(encoded, &result); err != nil {
		panic(err)
	}
	return
}
