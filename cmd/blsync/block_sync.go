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
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	lsync "github.com/ethereum/go-ethereum/beacon/light/sync"
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

type beaconBlockServer interface {
	request.RequestServer
	RequestBeaconBlock(blockRoot common.Hash, response func(*capella.BeaconBlock, error))
}

type beaconBlockSync struct {
	lock         sync.Mutex
	reqLock      request.MultiLock
	recentBlocks *lru.Cache[common.Hash, *capella.BeaconBlock]

	headUpdater                             *lsync.HeadUpdater
	validatedHead                           types.Header
	headBlock                               *capella.BeaconBlock // belongs to validatedHead (or nil)
	headBlockTrigger, prefetchHeaderTrigger *request.ModuleTrigger
}

func newBeaconBlockSyncer() *beaconBlockSync {
	return &beaconBlockSync{
		recentBlocks: lru.NewCache[common.Hash, *capella.BeaconBlock](10),
	}
}

// SetupModuleTriggers implements request.Module
func (s *beaconBlockSync) SetupModuleTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	s.reqLock.Trigger = trigger("beaconBlockSync", true)
	trigger("validatedHead", true)
	s.headBlockTrigger = trigger("headBlock", false)
	s.prefetchHeaderTrigger = trigger("prefetchHeader", false)
}

// Process implements request.Module
func (s *beaconBlockSync) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	validatedHead := env.ValidatedHead()
	if validatedHead == (types.Header{}) {
		return
	}
	if validatedHead != s.validatedHead {
		s.validatedHead = validatedHead
		s.headBlock = nil
		if block, ok := s.recentBlocks.Get(validatedHead.Hash()); ok {
			s.headBlock = block
			s.headBlockTrigger.Trigger()
		}
	}

	if !env.CanRequestNow() {
		return
	}
	if s.headBlock == nil && s.validatedHead != (types.Header{}) {
		s.tryRequestBlock(env, s.validatedHead.Hash(), false)
	}

	prefetchHead := env.PrefetchHead()
	if _, ok := s.recentBlocks.Get(prefetchHead); !ok {
		s.tryRequestBlock(env, prefetchHead, true)
	}
}

func (s *beaconBlockSync) getHeadBlock() *capella.BeaconBlock {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.headBlock
}

func (s *beaconBlockSync) tryRequestBlock(env *request.Environment, blockRoot common.Hash, prefetch bool) {
	if !s.reqLock.CanRequest(blockRoot) {
		return
	}
	env.TryRequest(blockRequest{
		beaconBlockSync: s,
		blockRoot:       blockRoot,
		prefetch:        prefetch,
	})
}

type blockRequest struct {
	*beaconBlockSync
	blockRoot common.Hash
	prefetch  bool
}

func (r blockRequest) CanSendTo(server *request.Server, moduleData *interface{}) (canSend bool, priority uint64) {
	if _, ok := server.RequestServer.(beaconBlockServer); !ok {
		return false, 0
	}
	if !r.prefetch {
		return true, 0
	}
	_, headRoot := server.LatestHead()
	return r.blockRoot == headRoot, 0
}

func (r blockRequest) SendTo(server *request.Server, moduleData *interface{}) {
	reqId := r.reqLock.Send(server, r.blockRoot)
	server.RequestServer.(beaconBlockServer).RequestBeaconBlock(r.blockRoot, func(block *capella.BeaconBlock, err error) {
		r.lock.Lock()
		defer r.lock.Unlock()

		r.reqLock.Returned(server, reqId, r.blockRoot)
		if block == nil || err != nil {
			server.Fail("error retrieving beacon block")
			return
		}
		r.recentBlocks.Add(r.blockRoot, block)
		if r.validatedHead.Hash() == r.blockRoot {
			r.headBlock = block
			r.headBlockTrigger.Trigger()
		}
	})
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
	client      *rpc.Client
	lock        sync.Mutex
	lastHead    common.Hash
	blockSync   *beaconBlockSync
	updating    bool
	selfTrigger *request.ModuleTrigger
}

// SetupModuleTriggers implements request.Module
func (s *engineApiUpdater) SetupModuleTriggers(trigger func(id string, subscribe bool) *request.ModuleTrigger) {
	trigger("headBlock", true)
	trigger("headState", true)
	s.selfTrigger = trigger("engineApiUpdater", true)
}

// Process implements request.Module
func (s *engineApiUpdater) Process(env *request.Environment) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.updating {
		return
	}
	headBlock := s.blockSync.getHeadBlock()
	if headBlock == nil {
		return
	}
	headRoot := common.Hash(headBlock.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
	if headRoot == s.lastHead {
		return
	}

	s.lastHead = headRoot
	execBlock, err := getExecBlock(headBlock)
	if err != nil {
		log.Error("Error extracting execution block from validated beacon block", "error", err)
		return
	}
	execRoot := execBlock.Hash()
	if s.client == nil { // dry run, no engine API specified
		log.Info("New execution block retrieved", "block number", execBlock.NumberU64(), "block hash", execRoot)
	} else {
		s.updating = true
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
			s.lock.Lock()
			s.updating = false
			s.selfTrigger.Trigger()
			s.lock.Unlock()
		}()
	}
}
