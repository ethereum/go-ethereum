// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
)

// replayAccessList installs one block's recorded post-state into the shadow
// tree: state opens at the parent's shadow root, the access list is applied
// without executing anything, and the commit runs with the canonical import's
// own arguments, so the shadow's leaves come from the same machinery
// execution uses. A state-identical block returns the parent root unchanged.
func replayAccessList(sdb state.Database, config *params.ChainConfig, parentRoot common.Hash, number *big.Int, time uint64, list *bal.BlockAccessList) (common.Hash, error) {
	statedb, err := state.New(parentRoot, sdb)
	if err != nil {
		return common.Hash{}, err
	}
	if err := statedb.ApplyBlockAccessList(list); err != nil {
		return common.Hash{}, err
	}
	return statedb.Commit(number.Uint64(), config.IsEIP158(number), config.IsCancun(number, time))
}

// replayBlock is one block's worth of replay input.
type replayBlock struct {
	number *big.Int
	time   uint64
	list   *bal.BlockAccessList
}

// replayRange folds a run of consecutive blocks and installs the range's end
// state in one commit. The fold may stop early at an account removal (see
// bal.Fold), so the number of blocks consumed is returned alongside the new
// shadow root.
func replayRange(sdb state.Database, config *params.ChainConfig, parentRoot common.Hash, blocks []replayBlock) (common.Hash, int, error) {
	lists := make([]*bal.BlockAccessList, len(blocks))
	for i, b := range blocks {
		lists[i] = b.list
	}
	folded, taken := bal.Fold(lists)
	if taken == 0 {
		return parentRoot, 0, nil
	}
	end := blocks[taken-1]
	root, err := replayAccessList(sdb, config, parentRoot, end.number, end.time, folded)
	if err != nil {
		return common.Hash{}, 0, err
	}
	return root, taken, nil
}
