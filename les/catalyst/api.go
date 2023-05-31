// Copyright 2022 The go-ethereum Authors
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

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// Register adds catalyst APIs to the light client.
func Register(stack *node.Node, backend *les.LightEthereum) error {
	log.Warn("Catalyst mode enabled", "protocol", "les")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       NewConsensusAPI(backend),
			Authenticated: true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	les *les.LightEthereum
}

// NewConsensusAPI creates a new consensus api for the given backend.
// The underlying blockchain needs to have a valid terminal total difficulty set.
func NewConsensusAPI(les *les.LightEthereum) *ConsensusAPI {
	if les.BlockChain().Config().TerminalTotalDifficulty == nil {
		log.Warn("Catalyst started without valid total difficulty")
	}
	return &ConsensusAPI{les: les}
}

// ForkchoiceUpdatedV1 has several responsibilities:
//
// We try to set our blockchain to the headBlock.
//
// If the method is called with an empty head block: we return success, which can be used
// to check if the catalyst mode is enabled.
//
// If the total difficulty was not reached: we return INVALID.
//
// If the finalizedBlockHash is set: we check if we have the finalizedBlockHash in our db,
// if not we start a sync.
//
// If there are payloadAttributes: we return an error since block creation is not
// supported in les mode.
func (api *ConsensusAPI) ForkchoiceUpdatedV1(heads engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	if heads.HeadBlockHash == (common.Hash{}) {
		log.Warn("Forkchoice requested update to zero hash")
		return engine.STATUS_INVALID, nil // TODO(karalabe): Why does someone send us this?
	}
	if err := api.checkTerminalTotalDifficulty(heads.HeadBlockHash); err != nil {
		if header := api.les.BlockChain().GetHeaderByHash(heads.HeadBlockHash); header == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return engine.STATUS_SYNCING, nil
		}
		return engine.STATUS_INVALID, err
	}
	// If the finalized block is set, check if it is in our blockchain
	if heads.FinalizedBlockHash != (common.Hash{}) {
		if header := api.les.BlockChain().GetHeaderByHash(heads.FinalizedBlockHash); header == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return engine.STATUS_SYNCING, nil
		}
	}
	// SetHead
	if err := api.setCanonical(heads.HeadBlockHash); err != nil {
		return engine.STATUS_INVALID, err
	}
	if payloadAttributes != nil {
		return engine.STATUS_INVALID, errors.New("not supported")
	}
	return api.validForkChoiceResponse(), nil
}

// GetPayloadV1 returns a cached payload by id. It's not supported in les mode.
func (api *ConsensusAPI) GetPayloadV1(payloadID engine.PayloadID) (*engine.ExecutableData, error) {
	return nil, engine.GenericServerError.With(errors.New("not supported in light client mode"))
}

// ExecutePayloadV1 creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) ExecutePayloadV1(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	block, err := engine.ExecutableDataToBlock(params)
	if err != nil {
		return api.invalid(), err
	}
	if !api.les.BlockChain().HasHeader(block.ParentHash(), block.NumberU64()-1) {
		/*
			TODO (MariusVanDerWijden) reenable once sync is merged
			if err := api.eth.Downloader().BeaconSync(api.eth.SyncMode(), block.Header()); err != nil {
				return SYNCING, err
			}
		*/
		// TODO (MariusVanDerWijden) we should return nil here not empty hash
		return engine.PayloadStatusV1{Status: engine.SYNCING, LatestValidHash: nil}, nil
	}
	parent := api.les.BlockChain().GetHeaderByHash(params.ParentHash)
	if parent == nil {
		return api.invalid(), fmt.Errorf("could not find parent %x", params.ParentHash)
	}
	td := api.les.BlockChain().GetTd(parent.Hash(), block.NumberU64()-1)
	ttd := api.les.BlockChain().Config().TerminalTotalDifficulty
	if td.Cmp(ttd) < 0 {
		return api.invalid(), fmt.Errorf("can not execute payload on top of block with low td got: %v threshold %v", td, ttd)
	}
	if err = api.les.BlockChain().InsertHeader(block.Header()); err != nil {
		return api.invalid(), err
	}
	if merger := api.les.Merger(); !merger.TDDReached() {
		merger.ReachTTD()
	}
	hash := block.Hash()
	return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &hash}, nil
}

func (api *ConsensusAPI) validForkChoiceResponse() engine.ForkChoiceResponse {
	currentHash := api.les.BlockChain().CurrentHeader().Hash()
	return engine.ForkChoiceResponse{
		PayloadStatus: engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &currentHash},
	}
}

// invalid returns a response "INVALID" with the latest valid hash set to the current head.
func (api *ConsensusAPI) invalid() engine.PayloadStatusV1 {
	currentHash := api.les.BlockChain().CurrentHeader().Hash()
	return engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &currentHash}
}

func (api *ConsensusAPI) checkTerminalTotalDifficulty(head common.Hash) error {
	// shortcut if we entered PoS already
	if api.les.Merger().PoSFinalized() {
		return nil
	}
	// make sure the parent has enough terminal total difficulty
	header := api.les.BlockChain().GetHeaderByHash(head)
	if header == nil {
		return errors.New("unknown header")
	}
	td := api.les.BlockChain().GetTd(header.Hash(), header.Number.Uint64())
	if td != nil && td.Cmp(api.les.BlockChain().Config().TerminalTotalDifficulty) < 0 {
		return errors.New("invalid ttd")
	}
	return nil
}

// setCanonical is called to perform a force choice.
func (api *ConsensusAPI) setCanonical(newHead common.Hash) error {
	log.Info("Setting head", "head", newHead)

	headHeader := api.les.BlockChain().CurrentHeader()
	if headHeader.Hash() == newHead {
		return nil
	}
	newHeadHeader := api.les.BlockChain().GetHeaderByHash(newHead)
	if newHeadHeader == nil {
		return errors.New("unknown header")
	}
	if err := api.les.BlockChain().SetCanonical(newHeadHeader); err != nil {
		return err
	}
	// Trigger the transition if it's the first `NewHead` event.
	if merger := api.les.Merger(); !merger.PoSFinalized() {
		merger.FinalizePoS()
	}
	return nil
}

// ExchangeTransitionConfigurationV1 checks the given configuration against
// the configuration of the node.
func (api *ConsensusAPI) ExchangeTransitionConfigurationV1(config engine.TransitionConfigurationV1) (*engine.TransitionConfigurationV1, error) {
	log.Trace("Engine API request received", "method", "ExchangeTransitionConfiguration", "ttd", config.TerminalTotalDifficulty)
	if config.TerminalTotalDifficulty == nil {
		return nil, errors.New("invalid terminal total difficulty")
	}

	ttd := api.les.BlockChain().Config().TerminalTotalDifficulty
	if ttd == nil || ttd.Cmp(config.TerminalTotalDifficulty.ToInt()) != 0 {
		log.Warn("Invalid TTD configured", "geth", ttd, "beacon", config.TerminalTotalDifficulty)
		return nil, fmt.Errorf("invalid ttd: execution %v consensus %v", ttd, config.TerminalTotalDifficulty)
	}

	if config.TerminalBlockHash != (common.Hash{}) {
		if hash := api.les.BlockChain().GetCanonicalHash(uint64(config.TerminalBlockNumber)); hash == config.TerminalBlockHash {
			return &engine.TransitionConfigurationV1{
				TerminalTotalDifficulty: (*hexutil.Big)(ttd),
				TerminalBlockHash:       config.TerminalBlockHash,
				TerminalBlockNumber:     config.TerminalBlockNumber,
			}, nil
		}
		return nil, errors.New("invalid terminal block hash")
	}

	return &engine.TransitionConfigurationV1{TerminalTotalDifficulty: (*hexutil.Big)(ttd)}, nil
}
