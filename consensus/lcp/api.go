// Copyright 2017 The go-ethereum Authors
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

package lcp

import (
	"github.com/pavelkrolevets/go-ethereum/common"
	"github.com/pavelkrolevets/go-ethereum/consensus"
	"github.com/pavelkrolevets/go-ethereum/core/types"
	"github.com/pavelkrolevets/go-ethereum/rpc"
	"github.com/pavelkrolevets/go-ethereum/trie"
	"math/big"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-authority scheme.
type API struct {
	chain consensus.ChainReader
	lcp   *LCP
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	epochTrie, err := types.NewEpochTrie(header.LCPContext.EpochHash, trie.NewDatabase(api.lcp.db))
	if err != nil {
		return nil, err
	}
	LCPContext := types.LCPContext{}
	LCPContext.SetEpoch(epochTrie)
	validators, err := LCPContext.GetValidators()
	if err != nil {
		return nil, err
	}
	return validators, nil
}

// GetConfirmedBlockNumber retrieves the latest irreversible block
func (api *API) GetConfirmedBlockNumber() (*big.Int, error) {
	var err error
	header := api.lcp.confirmedBlockHeader
	if header == nil {
		header, err = api.lcp.loadConfirmedBlockHeader(api.chain)
		if err != nil {
			return nil, err
		}
	}
	return header.Number, nil
}
