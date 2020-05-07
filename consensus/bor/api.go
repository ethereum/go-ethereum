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

package bor

import (
	"encoding/hex"
	"math"
	"math/big"
	"strconv"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/rpc"
	"github.com/xsleonard/go-merkle"
	"golang.org/x/crypto/sha3"
)

var (
	// MaxCheckpointLength is the maximum number of blocks that can be requested for constructing a checkpoint root hash
	MaxCheckpointLength = uint64(math.Pow(2, 15))
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-authority scheme.
type API struct {
	chain         consensus.ChainReader
	bor           *Bor
	rootHashCache *lru.ARCCache
}

// GetSnapshot retrieves the state snapshot at a given block.
func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.bor.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

// GetSnapshotAtHash retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.bor.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

// GetSigners retrieves the list of authorized signers at the specified block.
func (api *API) GetSigners(number *rpc.BlockNumber) ([]common.Address, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return the signers from its snapshot
	if header == nil {
		return nil, errUnknownBlock
	}
	snap, err := api.bor.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}

// GetSignersAtHash retrieves the list of authorized signers at the specified block.
func (api *API) GetSignersAtHash(hash common.Hash) ([]common.Address, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	snap, err := api.bor.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}

// GetCurrentProposer gets the current proposer
func (api *API) GetCurrentProposer() (common.Address, error) {
	snap, err := api.GetSnapshot(nil)
	if err != nil {
		return common.Address{}, err
	}
	return snap.ValidatorSet.GetProposer().Address, nil
}

// GetCurrentValidators gets the current validators
func (api *API) GetCurrentValidators() ([]*Validator, error) {
	snap, err := api.GetSnapshot(nil)
	if err != nil {
		return make([]*Validator, 0), err
	}
	return snap.ValidatorSet.Validators, nil
}

// GetRootHash returns the merkle root of the start to end block headers
func (api *API) GetRootHash(start uint64, end uint64) (string, error) {
	if err := api.initializeRootHashCache(); err != nil {
		return "", err
	}
	key := getRootHashKey(start, end)
	if root, known := api.rootHashCache.Get(key); known {
		return root.(string), nil
	}
	length := uint64(end - start + 1)
	if length > MaxCheckpointLength {
		return "", &MaxCheckpointLengthExceededError{start, end}
	}
	currentHeaderNumber := api.chain.CurrentHeader().Number.Uint64()
	if start > end || end > currentHeaderNumber {
		return "", &InvalidStartEndBlockError{start, end, currentHeaderNumber}
	}
	blockHeaders := make([]*types.Header, end-start+1)
	wg := new(sync.WaitGroup)
	concurrent := make(chan bool, 20)
	for i := start; i <= end; i++ {
		wg.Add(1)
		concurrent <- true
		go func(number uint64) {
			blockHeaders[number-start] = api.chain.GetHeaderByNumber(uint64(number))
			<-concurrent
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(concurrent)

	headers := make([][32]byte, nextPowerOfTwo(length))
	for i := 0; i < len(blockHeaders); i++ {
		blockHeader := blockHeaders[i]
		header := crypto.Keccak256(appendBytes32(
			blockHeader.Number.Bytes(),
			new(big.Int).SetUint64(blockHeader.Time).Bytes(),
			blockHeader.TxHash.Bytes(),
			blockHeader.ReceiptHash.Bytes(),
		))

		var arr [32]byte
		copy(arr[:], header)
		headers[i] = arr
	}

	tree := merkle.NewTreeWithOpts(merkle.TreeOptions{EnableHashSorting: false, DisableHashLeaves: true})
	if err := tree.Generate(convert(headers), sha3.NewLegacyKeccak256()); err != nil {
		return "", err
	}
	root := hex.EncodeToString(tree.Root().Hash)
	api.rootHashCache.Add(key, root)
	return root, nil
}

func (api *API) initializeRootHashCache() error {
	var err error
	if api.rootHashCache == nil {
		api.rootHashCache, err = lru.NewARC(10)
	}
	return err
}

func getRootHashKey(start uint64, end uint64) string {
	return strconv.FormatUint(start, 10) + "-" + strconv.FormatUint(end, 10)
}
