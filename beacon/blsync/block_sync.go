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

package blsync

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

// beaconBlockSync implements request.Module; it fetches the beacon blocks belonging
// to the validated and prefetch heads.
type beaconBlockSync struct {
	recentBlocks *lru.Cache[common.Hash, *capella.BeaconBlock]
	locked       map[common.Hash]request.ServerAndID
	serverHeads  map[request.Server]common.Hash
	headTracker  headTracker

	lastHeadInfo  types.HeadInfo
	chainHeadFeed *event.Feed
}

type headTracker interface {
	PrefetchHead() types.HeadInfo
	ValidatedHead() (types.SignedHeader, bool)
	ValidatedFinality() (types.FinalityUpdate, bool)
}

// newBeaconBlockSync returns a new beaconBlockSync.
func newBeaconBlockSync(headTracker headTracker, chainHeadFeed *event.Feed) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:   headTracker,
		chainHeadFeed: chainHeadFeed,
		recentBlocks:  lru.NewCache[common.Hash, *capella.BeaconBlock](10),
		locked:        make(map[common.Hash]request.ServerAndID),
		serverHeads:   make(map[request.Server]common.Hash),
	}
}

// Process implements request.Module.
func (s *beaconBlockSync) Process(requester request.Requester, events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			blockRoot := common.Hash(req.(sync.ReqBeaconBlock))
			if resp != nil {
				s.recentBlocks.Add(blockRoot, resp.(*capella.BeaconBlock))
			}
			if s.locked[blockRoot] == sid {
				delete(s.locked, blockRoot)
			}
		case sync.EvNewHead:
			s.serverHeads[event.Server] = event.Data.(types.HeadInfo).BlockRoot
		case request.EvUnregistered:
			delete(s.serverHeads, event.Server)
		}
	}
	s.updateEventFeed()
	// request validated head block if unavailable and not yet requested
	if vh, ok := s.headTracker.ValidatedHead(); ok {
		s.tryRequestBlock(requester, vh.Header.Hash(), false)
	}
	// request prefetch head if the given server has announced it
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) {
		s.tryRequestBlock(requester, prefetchHead, true)
	}
}

func (s *beaconBlockSync) tryRequestBlock(requester request.Requester, blockRoot common.Hash, needSameHead bool) {
	if _, ok := s.recentBlocks.Get(blockRoot); ok {
		return
	}
	if _, ok := s.locked[blockRoot]; ok {
		return
	}
	for _, server := range requester.CanSendTo() {
		if needSameHead && (s.serverHeads[server] != blockRoot) {
			continue
		}
		id := requester.Send(server, sync.ReqBeaconBlock(blockRoot))
		s.locked[blockRoot] = request.ServerAndID{Server: server, ID: id}
		return
	}
}

func blockHeadInfo(block *capella.BeaconBlock) types.HeadInfo {
	if block == nil {
		return types.HeadInfo{}
	}
	return types.HeadInfo{Slot: uint64(block.Slot), BlockRoot: beaconBlockHash(block)}
}

// beaconBlockHash calculates the hash of a beacon block.
func beaconBlockHash(beaconBlock *capella.BeaconBlock) common.Hash {
	return common.Hash(beaconBlock.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
}

// getExecBlock extracts the execution block from the beacon block's payload.
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
		return execBlock, fmt.Errorf("Sanity check failed, payload hash does not match (expected %x, got %x)", common.Hash(payload.BlockHash), execBlockHash)
	}
	return execBlock, nil
}

func (s *beaconBlockSync) updateEventFeed() {
	head, ok := s.headTracker.ValidatedHead()
	if !ok {
		return
	}
	finality, ok := s.headTracker.ValidatedFinality() //TODO fetch directly if subscription does not deliver
	if !ok || head.Header.Epoch() != finality.Attested.Header.Epoch() {
		return
	}
	validatedHead := head.Header.Hash()
	headBlock, ok := s.recentBlocks.Get(validatedHead)
	if !ok {
		return
	}
	headInfo := blockHeadInfo(headBlock)
	if headInfo == s.lastHeadInfo {
		return
	}
	s.lastHeadInfo = headInfo
	// new head block and finality info available; extract executable data and send event to feed
	execBlock, err := getExecBlock(headBlock)
	if err != nil {
		log.Error("Error extracting execution block from validated beacon block", "error", err)
		return
	}
	s.chainHeadFeed.Send(types.ChainHeadEvent{
		HeadBlock: engine.BlockToExecutableData(execBlock, nil, nil).ExecutionPayload,
		Finalized: common.Hash(finality.Finalized.PayloadHeader.BlockHash),
	})
}
