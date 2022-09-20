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

package main

import (
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	ctypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

const reverseSyncHeaders = 128

type beaconBlockSync struct {
	recentBlocks  *lru.Cache[common.Hash, *capella.BeaconBlock]
	validatedHead types.Header
	pending       map[common.Hash]struct{}
	serverHeads   map[request.Server]common.Hash
	headTracker   *light.HeadTracker
}

func newBeaconBlockSyncer(headTracker *light.HeadTracker) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:  headTracker,
		recentBlocks: lru.NewCache[common.Hash, *capella.BeaconBlock](10),
		pending:      make(map[common.Hash]struct{}),
		serverHeads:  make(map[request.Server]common.Hash),
	}
}

// Process implements request.Module
func (s *beaconBlockSync) Process(tracker *request.RequestTracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) (trigger bool) {
	s.validatedHead = s.headTracker.ValidatedHead().Header
	if s.validatedHead == (types.Header{}) {
		return false
	}

	// iterate events and add valid responses to recentBlocks
	for _, event := range requestEvents {
		blockRoot := common.Hash(event.Request.(sync.ReqBeaconBlock))
		if event.Response != nil {
			block := event.Response.(*capella.BeaconBlock)
			s.recentBlocks.Add(blockRoot, block)
			if blockRoot == s.validatedHead.Hash() {
				trigger = true
			}
		}
		if event.Timeout != event.Finalized {
			// unlock if timed out or returned with an invalid response without
			// previously being unlocked by a timeout
			delete(s.pending, blockRoot)
		}
	}

	// update server heads
	for _, event := range serverEvents {
		switch event.Type {
		case sync.EvNewHead:
			s.serverHeads[event.Server] = event.Data.(types.HeadInfo).BlockRoot
		case request.EvUnregistered:
			delete(s.serverHeads, event.Server)
		}
	}

	// start new requests if necessary
	s.tryRequestBlock(tracker, s.validatedHead.Hash(), false)
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) {
		s.tryRequestBlock(tracker, prefetchHead, true)
	}
	return
}

// belongs to validatedHead (or nil)
func (s *beaconBlockSync) getHeadBlock() *capella.BeaconBlock {
	block, _ := s.recentBlocks.Get(s.validatedHead.Hash())
	return block
}

func (s *beaconBlockSync) tryRequestBlock(tracker *request.RequestTracker, blockRoot common.Hash, prefetch bool) {
	if _, ok := s.recentBlocks.Get(blockRoot); ok {
		return
	}
	if _, ok := s.pending[blockRoot]; ok {
		return
	}
	if _, request := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
		if prefetch && s.serverHeads[server] != blockRoot {
			// when requesting a not yet validated head, request it from someone
			// who has announced it already
			return nil, 0
		}
		return sync.ReqBeaconBlock(blockRoot), 0
	}); request != nil {
		s.pending[blockRoot] = struct{}{}
	}
}

func getExecBlock(beaconBlock *capella.BeaconBlock) (*ctypes.Block, error) {
	payload := &beaconBlock.Body.ExecutionPayload
	txs := make([]*ctypes.Transaction, len(payload.Transactions))
	for i, opaqueTx := range payload.Transactions {
		var tx ctypes.Transaction
		if err := tx.UnmarshalBinary(opaqueTx); err != nil {
			return nil, fmt.Errorf("failed to parse tx %d: %v", i, err)
		}
		txs[i] = &tx
	}
	withdrawals := make([]*ctypes.Withdrawal, len(payload.Withdrawals))
	for i, w := range payload.Withdrawals {
		withdrawals[i] = &ctypes.Withdrawal{
			Index:     uint64(w.Index),
			Validator: uint64(w.ValidatorIndex),
			Address:   common.Address(w.Address),
			Amount:    uint64(w.Amount),
		}
	}
	wroot := ctypes.DeriveSha(ctypes.Withdrawals(withdrawals), trie.NewStackTrie(nil))
	execHeader := &ctypes.Header{
		ParentHash:      common.Hash(payload.ParentHash),
		UncleHash:       ctypes.EmptyUncleHash,
		Coinbase:        common.Address(payload.FeeRecipient),
		Root:            common.Hash(payload.StateRoot),
		TxHash:          ctypes.DeriveSha(ctypes.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:     common.Hash(payload.ReceiptsRoot),
		Bloom:           ctypes.Bloom(payload.LogsBloom),
		Difficulty:      common.Big0,
		Number:          new(big.Int).SetUint64(uint64(payload.BlockNumber)),
		GasLimit:        uint64(payload.GasLimit),
		GasUsed:         uint64(payload.GasUsed),
		Time:            uint64(payload.Timestamp),
		Extra:           []byte(payload.ExtraData),
		MixDigest:       common.Hash(payload.PrevRandao), // reused in merge
		Nonce:           ctypes.BlockNonce{},             // zero
		BaseFee:         (*uint256.Int)(&payload.BaseFeePerGas).ToBig(),
		WithdrawalsHash: &wroot,
	}
	execBlock := ctypes.NewBlockWithHeader(execHeader).WithBody(txs, nil).WithWithdrawals(withdrawals)
	if execBlockHash := execBlock.Hash(); execBlockHash != common.Hash(payload.BlockHash) {
		return nil, fmt.Errorf("Sanity check failed, payload hash does not match (expected %x, got %x)", common.Hash(payload.BlockHash), execBlockHash)
	}
	return execBlock, nil
}

type engineApiUpdater struct {
	client    *rpc.Client
	trigger   func()
	lastHead  common.Hash
	blockSync *beaconBlockSync
	updating  uint32
}

// Process implements request.Module
func (s *engineApiUpdater) Process(tracker *request.RequestTracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) bool {
	if atomic.LoadUint32(&s.updating) == 1 {
		return false
	}
	headBlock := s.blockSync.getHeadBlock()
	if headBlock == nil {
		return false
	}
	headRoot := common.Hash(headBlock.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
	if headRoot == s.lastHead {
		return false
	}

	s.lastHead = headRoot
	execBlock, err := getExecBlock(headBlock)
	if err != nil {
		log.Error("Error extracting execution block from validated beacon block", "error", err)
		return false
	}
	execRoot := execBlock.Hash()
	if s.client == nil { // dry run, no engine API specified
		log.Info("New execution block retrieved", "block number", execBlock.NumberU64(), "block hash", execRoot)
	} else {
		atomic.StoreUint32(&s.updating, 1)
		go func() {
			if status, err := callNewPayloadV2(s.client, execBlock); err == nil {
				log.Info("Successful NewPayload", "block number", execBlock.NumberU64(), "block hash", execRoot, "status", status)
			} else {
				log.Error("Failed NewPayload", "block number", execBlock.NumberU64(), "block hash", execRoot, "error", err)
			}
			if status, err := callForkchoiceUpdatedV1(s.client, execRoot, common.Hash{}); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", execRoot, "status", status)
			} else {
				log.Error("Failed ForkchoiceUpdated", "head", execRoot, "error", err)
			}
			atomic.StoreUint32(&s.updating, 0)
			s.trigger()
		}()
	}
	return false
}
