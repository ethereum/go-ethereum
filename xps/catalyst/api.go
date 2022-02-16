// Copyright 2020 The go-ethereum Authors
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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

// Package catalyst implements the temporary xps1/xps2 RPC integration.

package catalyst

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/xpaymentsorg/go-xpayments/common"
	"github.com/xpaymentsorg/go-xpayments/core/beacon"
	"github.com/xpaymentsorg/go-xpayments/core/types"
	"github.com/xpaymentsorg/go-xpayments/log"
	"github.com/xpaymentsorg/go-xpayments/node"
	"github.com/xpaymentsorg/go-xpayments/rpc"
	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/core/beacon"
	// "github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/eth"
	// "github.com/ethereum/go-ethereum/log"
	// "github.com/ethereum/go-ethereum/node"
	// "github.com/ethereum/go-ethereum/rpc"
)

// Register adds catalyst APIs to the full node.
func Register(stack *node.Node, backend *xps.xPayments) error {
	log.Warn("Catalyst mode enabled", "protocol", "xps")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "engine",
			Version:   "1.0",
			Service:   NewConsensusAPI(backend),
			Public:    true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	xps            *xps.xPayments
	preparedBlocks *payloadQueue // preparedBlocks caches payloads (*ExecutableDataV1) by payload ID (PayloadID)
}

// NewConsensusAPI creates a new consensus api for the given backend.
// The underlying blockchain needs to have a valid terminal total difficulty set.
func NewConsensusAPI(xps *xps.xPayments) *ConsensusAPI {
	if xps.BlockChain().Config().TerminalTotalDifficulty == nil {
		panic("Catalyst started without valid total difficulty")
	}
	return &ConsensusAPI{
		xps:            xps,
		preparedBlocks: newPayloadQueue(),
	}
}

// ForkchoiceUpdatedV1 has several responsibilities:
// If the method is called with an empty head block:
// 		we return success, which can be used to check if the catalyst mode is enabled
// If the total difficulty was not reached:
// 		we return INVALID
// If the finalizedBlockHash is set:
// 		we check if we have the finalizedBlockHash in our db, if not we start a sync
// We try to set our blockchain to the headBlock
// If there are payloadAttributes:
// 		we try to assemble a block with the payloadAttributes and return its payloadID
func (api *ConsensusAPI) ForkchoiceUpdatedV1(heads beacon.ForkchoiceStateV1, payloadAttributes *beacon.PayloadAttributesV1) (beacon.ForkChoiceResponse, error) {
	log.Trace("Engine API request received", "method", "ForkChoiceUpdated", "head", heads.HeadBlockHash, "finalized", heads.FinalizedBlockHash, "safe", heads.SafeBlockHash)
	if heads.HeadBlockHash == (common.Hash{}) {
		return beacon.ForkChoiceResponse{Status: beacon.SUCCESS.Status, PayloadID: nil}, nil
	}
	if err := api.checkTerminalTotalDifficulty(heads.HeadBlockHash); err != nil {
		if block := api.xps.BlockChain().GetBlockByHash(heads.HeadBlockHash); block == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return beacon.SYNCING, nil
		}
		return beacon.INVALID, err
	}
	// If the finalized block is set, check if it is in our blockchain
	if heads.FinalizedBlockHash != (common.Hash{}) {
		if block := api.xps.BlockChain().GetBlockByHash(heads.FinalizedBlockHash); block == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return beacon.SYNCING, nil
		}
	}
	// SetHead
	if err := api.setHead(heads.HeadBlockHash); err != nil {
		return beacon.INVALID, err
	}
	// Assemble block (if needed). It only works for full node.
	if payloadAttributes != nil {
		data, err := api.assembleBlock(heads.HeadBlockHash, payloadAttributes)
		if err != nil {
			return beacon.INVALID, err
		}
		id := computePayloadId(heads.HeadBlockHash, payloadAttributes)
		api.preparedBlocks.put(id, data)
		log.Info("Created payload", "payloadID", id)
		return beacon.ForkChoiceResponse{Status: beacon.SUCCESS.Status, PayloadID: &id}, nil
	}
	return beacon.ForkChoiceResponse{Status: beacon.SUCCESS.Status, PayloadID: nil}, nil
}

// GetPayloadV1 returns a cached payload by id.
func (api *ConsensusAPI) GetPayloadV1(payloadID beacon.PayloadID) (*beacon.ExecutableDataV1, error) {
	log.Trace("Engine API request received", "method", "GetPayload", "id", payloadID)
	data := api.preparedBlocks.get(payloadID)
	if data == nil {
		return nil, &beacon.UnknownPayload
	}
	return data, nil
}

// ExecutePayloadV1 creates an Xps1 block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) ExecutePayloadV1(params beacon.ExecutableDataV1) (beacon.ExecutePayloadResponse, error) {
	log.Trace("Engine API request received", "method", "ExecutePayload", params.BlockHash, "number", params.Number)
	block, err := beacon.ExecutableDataToBlock(params)
	if err != nil {
		return api.invalid(), err
	}
	if !api.xps.BlockChain().HasBlock(block.ParentHash(), block.NumberU64()-1) {
		/*
			TODO (MariusVanDerWijden) reenable once sync is merged
			if err := api.xps.Downloader().BeaconSync(api.xps.SyncMode(), block.Header()); err != nil {
				return SYNCING, err
			}
		*/
		// TODO (MariusVanDerWijden) we should return nil here not empty hash
		return beacon.ExecutePayloadResponse{Status: beacon.SYNCING.Status, LatestValidHash: common.Hash{}}, nil
	}
	parent := api.xps.BlockChain().GetBlockByHash(params.ParentHash)
	td := api.xps.BlockChain().GetTd(parent.Hash(), block.NumberU64()-1)
	ttd := api.xps.BlockChain().Config().TerminalTotalDifficulty
	if td.Cmp(ttd) < 0 {
		return api.invalid(), fmt.Errorf("can not execute payload on top of block with low td got: %v threshold %v", td, ttd)
	}
	log.Trace("Inserting block without head", "hash", block.Hash(), "number", block.Number)
	if err := api.xps.BlockChain().InsertBlockWithoutSetHead(block); err != nil {
		return api.invalid(), err
	}

	if merger := api.xps.Merger(); !merger.TDDReached() {
		merger.ReachTTD()
	}
	return beacon.ExecutePayloadResponse{Status: beacon.VALID.Status, LatestValidHash: block.Hash()}, nil
}

// computePayloadId computes a pseudo-random payloadid, based on the parameters.
func computePayloadId(headBlockHash common.Hash, params *beacon.PayloadAttributesV1) beacon.PayloadID {
	// Hash
	hasher := sha256.New()
	hasher.Write(headBlockHash[:])
	binary.Write(hasher, binary.BigEndian, params.Timestamp)
	hasher.Write(params.Random[:])
	hasher.Write(params.SuggestedFeeRecipient[:])
	var out beacon.PayloadID
	copy(out[:], hasher.Sum(nil)[:8])
	return out
}

// invalid returns a response "INVALID" with the latest valid hash set to the current head.
func (api *ConsensusAPI) invalid() beacon.ExecutePayloadResponse {
	return beacon.ExecutePayloadResponse{Status: beacon.INVALID.Status, LatestValidHash: api.xps.BlockChain().CurrentHeader().Hash()}
}

// assembleBlock creates a new block and returns the "execution
// data" required for beacon clients to process the new block.
func (api *ConsensusAPI) assembleBlock(parentHash common.Hash, params *beacon.PayloadAttributesV1) (*beacon.ExecutableDataV1, error) {
	log.Info("Producing block", "parentHash", parentHash)
	block, err := api.xps.Miner().GetSealingBlock(parentHash, params.Timestamp, params.SuggestedFeeRecipient, params.Random)
	if err != nil {
		return nil, err
	}
	return beacon.BlockToExecutableData(block), nil
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *ConsensusAPI) insertTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		api.xps.TxPool().AddLocal(tx)
	}
	return nil
}

func (api *ConsensusAPI) checkTerminalTotalDifficulty(head common.Hash) error {
	// shortcut if we entered PoS already
	if api.xps.Merger().PoSFinalized() {
		return nil
	}
	// make sure the parent has enough terminal total difficulty
	newHeadBlock := api.xps.BlockChain().GetBlockByHash(head)
	if newHeadBlock == nil {
		return &beacon.GenericServerError
	}
	td := api.xps.BlockChain().GetTd(newHeadBlock.Hash(), newHeadBlock.NumberU64())
	if td != nil && td.Cmp(api.xps.BlockChain().Config().TerminalTotalDifficulty) < 0 {
		return &beacon.InvalidTB
	}
	return nil
}

// setHead is called to perform a force choice.
func (api *ConsensusAPI) setHead(newHead common.Hash) error {
	log.Info("Setting head", "head", newHead)
	headBlock := api.xps.BlockChain().CurrentBlock()
	if headBlock.Hash() == newHead {
		return nil
	}
	newHeadBlock := api.xps.BlockChain().GetBlockByHash(newHead)
	if newHeadBlock == nil {
		return &beacon.GenericServerError
	}
	if err := api.xps.BlockChain().SetChainHead(newHeadBlock); err != nil {
		return err
	}
	// Trigger the transition if it's the first `NewHead` event.
	if merger := api.xps.Merger(); !merger.PoSFinalized() {
		merger.FinalizePoS()
	}
	// TODO (MariusVanDerWijden) are we really synced now?
	api.xps.SetSynced()
	return nil
}
